package launcher

import (
	"os"
	"path/filepath"
	"pika/internal/config"
	"reflect"
	"testing"
)

func TestSystemActionsWithoutPersonalCommands(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"cinnamon-session-quit", "cinnamon-screensaver-command"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nexit 99\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	cfg := config.Defaults()
	cfg.Search.IncludeCommands = false
	cfg.Watcher.Enabled = false
	s := New(cfg, filepath.Join(dir, "config.toml"), filepath.Join(dir, "state.db"))
	s.Start()
	defer s.Close()
	eventually(t, func() bool { return s.Status().Version > 0 && !s.Status().Indexing })
	for query, id := range map[string]string{"lock": "lock", "logout": "logout", "log out": "logout", "shutdown": "shutdown", "shut down": "shutdown", "tat may": "shutdown"} {
		results := s.Search(query, "all", 1).Results
		if len(results) == 0 || results[0].ID != "system:"+id {
			t.Fatalf("%q: %+v", query, results)
		}
		d, err := s.Details("system:" + id)
		if err != nil || d.Kind != "system" || d.Opener == "" {
			t.Fatalf("details: %+v %v", d, err)
		}
	}
	if s.Status().Commands != 0 || s.Status().System != 3 {
		t.Fatal(s.Status())
	}
}
func TestSystemCommandsKeepNativePrompts(t *testing.T) {
	for action, want := range map[string][]string{
		"lock":     {"cinnamon-screensaver-command", "--lock"},
		"logout":   {"cinnamon-session-quit", "--logout"},
		"shutdown": {"cinnamon-session-quit", "--power-off"},
	} {
		cmd, err := SystemCommand(action)
		if err != nil || !reflect.DeepEqual(cmd.Args, want) {
			t.Fatalf("%s: %v %v", action, cmd, err)
		}
	}
	if _, err := SystemCommand("shutdown --force"); err == nil {
		t.Fatal("unknown action accepted")
	}
}
