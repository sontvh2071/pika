package launcher

import (
	"context"
	"os"
	"path/filepath"
	"pika/internal/catalog"
	"pika/internal/config"
	"reflect"
	"testing"
)

func TestWorkspaceRouting(t *testing.T) {
	cfg := config.Open{WorkspaceRoot: "/home/test/workspace", WorkspaceExecutable: "code"}
	for _, tc := range []struct {
		path   string
		editor bool
	}{
		{"/home/test/workspace", true}, {"/home/test/workspace/me/project", true},
		{"/home/test/workspace/a file ; literal.txt", true}, {"/home/test/workspace/../Downloads", false},
		{"/home/test/workspace-backup", false}, {"/home/test/Downloads", false},
		{"/home/test/Pictures/photo.jpg", false}, {"/home/test/Desktop", false},
	} {
		cmd := FileCommand(cfg, tc.path)
		want := []string{"xdg-open", tc.path}
		if tc.editor {
			want = []string{"code", "--reuse-window", "--", tc.path}
		}
		if !reflect.DeepEqual(cmd.Args, want) {
			t.Fatalf("%s: %v want %v", tc.path, cmd.Args, want)
		}
	}
	cfg.WorkspaceRoot = ""
	if cmd := FileCommand(cfg, "/home/test/workspace"); cmd.Args[0] != "xdg-open" {
		t.Fatal(cmd.Args)
	}
}
func TestDetailsAndDisabledCommands(t *testing.T) {
	cfg := config.Defaults()
	cfg.Search.IncludeCommands = false
	cfg.Index.Roots = []string{t.TempDir()}
	cfg.Watcher.Enabled = false
	cfg.Commands = []config.Command{{ID: "unused", Name: "Unused", Executable: "unused"}}
	cfg.Open.WorkspaceRoot = cfg.Index.Roots[0]
	s := New(cfg, filepath.Join(t.TempDir(), "config.toml"), filepath.Join(t.TempDir(), "state.db"))
	s.Start()
	defer s.Close()
	eventually(t, func() bool { return s.Status().Version > 0 && !s.Status().Indexing })
	if s.Status().Commands != 0 || len(s.Search(">", "all", 1).Results) != 0 {
		t.Fatal("disabled commands indexed")
	}
	id := "directory:" + cfg.Index.Roots[0]
	d, err := s.Details(id)
	if err != nil || d.Kind != "directory" || d.Path != cfg.Index.Roots[0] || d.Opener != "VS Code" || d.Version != "" {
		t.Fatalf("%+v %v", d, err)
	}
	if _, err = s.Details("missing"); err == nil {
		t.Fatal("missing result accepted")
	}
}
func TestDebVersion(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
case "$1" in
 --search) printf 'test-app:amd64: %s\n' "$2" ;;
 --show) test "$3" = 'test-app:amd64' || exit 1; printf '2.4.1-1' ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "dpkg-query"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if got := debVersion(context.Background(), "/usr/share/applications/test.desktop"); got != "2.4.1-1" {
		t.Fatal(got)
	}
}
func TestDetailsUsePackageVersionNotDesktopSpec(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.desktop")
	if err := os.WriteFile(path, []byte("[Desktop Entry]\nType=Application\nName=Test\nVersion=1.0\nExec=do-not-launch --version\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c, ok, err := catalog.DesktopEntry(path, "test.desktop", "")
	if err != nil || !ok {
		t.Fatal(err)
	}
	if c.Path != path || c.VersionHint != "" {
		t.Fatal(c)
	}
	t.Setenv("PATH", dir) // no package database helper: don't guess or launch Exec.
	s := New(config.Defaults(), "", filepath.Join(dir, "state.db"))
	defer s.Close()
	s.publish([]catalog.Candidate{c})
	d, err := s.Details(c.ID)
	if err != nil || d.Path != path || d.Version != "" {
		t.Fatalf("%+v %v", d, err)
	}
}
