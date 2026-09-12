package codexusage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

func executable() (string, error) {
	// Prefer the CLI shipped with the desktop app whose quota is being shown.
	for _, path := range []string{"/usr/lib/chatgpt/resources/codex", "/opt/Codex/resources/codex"} {
		if info, err := os.Stat(path); err == nil && !info.IsDir() && info.Mode()&0111 != 0 {
			return path, nil
		}
	}
	return exec.LookPath("codex")
}
func fetchUsage(parent context.Context) ([]byte, error) {
	path, err := executable()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(parent, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "app-server", "--stdio", "-c", "analytics.enabled=false")
	cmd.Dir, _ = os.UserHomeDir()
	cmd.Stderr = io.Discard
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		input.Close()
		output.Close()
		return nil, err
	}
	defer func() { input.Close(); cancel(); _ = cmd.Wait() }()
	return exchange(input, output)
}
func exchange(input io.Writer, output io.Reader) ([]byte, error) {
	enc := json.NewEncoder(input)
	scan := bufio.NewScanner(output)
	scan.Buffer(make([]byte, 4096), 4*1024*1024)
	receive := func(want int) (json.RawMessage, error) {
		for scan.Scan() {
			var msg struct {
				ID     json.RawMessage `json:"id"`
				Method string          `json:"method"`
				Result json.RawMessage `json:"result"`
				Error  json.RawMessage `json:"error"`
			}
			if err := json.Unmarshal(scan.Bytes(), &msg); err != nil {
				return nil, fmt.Errorf("Invalid app-server message")
			}
			if msg.Method != "" {
				// No interactive server request is authorized for a quota-only reader.
				if len(msg.ID) > 0 && string(msg.ID) != "null" {
					if err := enc.Encode(map[string]any{"id": msg.ID, "error": map[string]any{"code": -32601, "message": "Pika only reads account quotas"}}); err != nil {
						return nil, err
					}
				}
				continue
			}
			if string(msg.ID) != fmt.Sprint(want) {
				continue
			}
			if len(msg.Error) > 0 && string(msg.Error) != "null" {
				return nil, fmt.Errorf("Codex rejected the quota request")
			}
			if len(msg.Result) == 0 || string(msg.Result) == "null" {
				return nil, fmt.Errorf("Codex returned no quota data")
			}
			return msg.Result, nil
		}
		if err := scan.Err(); err != nil {
			return nil, err
		}
		return nil, io.ErrUnexpectedEOF
	}
	if err := enc.Encode(map[string]any{"id": 1, "method": "initialize", "params": map[string]any{"clientInfo": map[string]any{"name": "pika_usage", "title": "Pika usage viewer", "version": "0.1.0"}}}); err != nil {
		return nil, err
	}
	if _, err := receive(1); err != nil {
		return nil, err
	}
	if err := enc.Encode(map[string]any{"method": "initialized"}); err != nil {
		return nil, err
	}
	if err := enc.Encode(map[string]any{"id": 2, "method": "account/rateLimits/read"}); err != nil {
		return nil, err
	}
	return receive(2)
}
