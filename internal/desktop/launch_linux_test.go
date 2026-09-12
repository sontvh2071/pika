//go:build linux

package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLaunchKeepsLiveParentAndDesktopArguments(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "probe script.sh")
	output := filepath.Join(dir, "result")
	body := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$PPID\" \"$1\" > '%s'\n", output)
	if err := os.WriteFile(script, []byte(body), 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "probe.desktop")
	entry := fmt.Sprintf("[Desktop Entry]\nType=Application\nName=Launch probe\nExec=/bin/sh \"%s\" \"literal ; not a command\"\nTerminal=false\n", script)
	if err := os.WriteFile(path, []byte(entry), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Launch(path); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		content, err := os.ReadFile(output)
		if err == nil && strings.Contains(string(content), "\n") {
			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			if len(lines) != 2 || lines[0] != strconv.Itoa(os.Getpid()) || lines[1] != "literal ; not a command" {
				t.Fatalf("child lost Pika parent or argv: %q", content)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("desktop child did not produce its parent/argv probe")
}
func TestInvalidDesktopEntry(t *testing.T) {
	for _, path := range []string{"relative.desktop", filepath.Join(t.TempDir(), "missing.desktop")} {
		if err := Launch(path); err == nil {
			t.Fatal("invalid desktop accepted", path)
		}
	}
}
