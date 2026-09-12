// Read-only diagnostic: index the configured sources in a temporary state DB,
// inspect common home folders and VS Code metadata, and never launch a candidate.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"pika/internal/config"
	"pika/internal/launcher"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	cfg, err := config.Load(config.Path(), false)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "pika-profile-inspect-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	cfg.Watcher.Enabled = false
	service := launcher.New(cfg, config.Path(), filepath.Join(tmp, "state.db"))
	service.Start()
	defer service.Close()
	deadline := time.Now().Add(60 * time.Second)
	for service.Status().Version == 0 || service.Status().Indexing {
		if time.Now().After(deadline) {
			return fmt.Errorf("index did not finish within 60 seconds")
		}
		time.Sleep(100 * time.Millisecond)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	details := []launcher.Details{}
	for _, name := range []string{"Downloads", "Desktop", "Pictures", "workspace"} {
		path := filepath.Join(home, name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		found := false
		for _, r := range service.Search(name, "file", 1).Results {
			if r.Path == path {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("expected folder not found in search: %s", path)
		}
		d, err := service.Details("directory:" + path)
		if err != nil {
			return err
		}
		details = append(details, d)
	}
	for _, r := range service.Search("Visual Studio Code", "app", 2).Results {
		d, err := service.Details(r.ID)
		if err == nil {
			details = append(details, d)
		}
	}
	for query, id := range map[string]string{"lock": "system:lock", "logout": "system:logout", "shutdown": "system:shutdown"} {
		results := service.Search(query, "all", 3).Results
		if len(results) == 0 || results[0].ID != id {
			return fmt.Errorf("system keyword %q did not resolve first: %+v", query, results)
		}
		d, err := service.Details(id)
		if err != nil {
			return err
		}
		details = append(details, d)
	}
	for _, excluded := range cfg.Index.ExcludePaths {
		path := filepath.Clean(config.Expand(excluded))
		if _, err := service.Details("directory:" + path); err == nil {
			return fmt.Errorf("excluded directory remains indexed: %s", path)
		}
	}
	result := struct {
		Status  launcher.Status    `json:"status"`
		Details []launcher.Details `json:"details"`
	}{service.Status(), details}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
