package launcher

import (
	"os"
	"path/filepath"
	"pika/internal/config"
	"sync"
	"testing"
	"time"
)

func eventually(t *testing.T, f func() bool) {
	t.Helper()
	until := time.Now().Add(12 * time.Second)
	for time.Now().Before(until) {
		if f() {
			return
		}
		time.Sleep(40 * time.Millisecond)
	}
	t.Fatal("condition did not become true")
}
func TestServiceRefreshExecuteReload(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "files")
	if e := os.Mkdir(root, 0700); e != nil {
		t.Fatal(e)
	}
	configPath := filepath.Join(dir, "config.toml")
	c, e := config.Load(configPath, true)
	if e != nil {
		t.Fatal(e)
	}
	c.Index.Roots = []string{root}
	marker := filepath.Join(dir, "literal ; touch not-a-shell")
	c.Commands = []config.Command{{ID: "test", Name: "Test command", Executable: "/usr/bin/touch", Args: []string{marker}}}
	s := New(c, configPath, filepath.Join(dir, "state.db"))
	s.Start()
	defer s.Close()
	eventually(t, func() bool { return s.Status().Version > 0 && !s.Status().Indexing })
	if e = s.Execute("command:test"); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(marker); e != nil {
		t.Fatal("argv was not passed literally", e)
	}
	if e = s.Execute("invalid"); e == nil {
		t.Fatal("executed invalid id")
	}
	file := filepath.Join(root, "new-document.txt")
	if e = os.WriteFile(file, []byte("content"), 0600); e != nil {
		t.Fatal(e)
	}
	eventually(t, func() bool { return len(s.Search("new-document", "file", 1).Results) > 0 })
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 30; j++ {
				s.Search("doc", "all", uint64(j))
				s.record("command:test")
				s.Reindex()
			}
		}()
	}
	wg.Wait()
	if e = os.WriteFile(configPath, []byte("broken = ["), 0600); e != nil {
		t.Fatal(e)
	}
	if e = s.Reload(); e == nil {
		t.Fatal("bad config accepted")
	}
	if len(s.Config().Commands) != 1 {
		t.Fatal("failed reload changed config")
	}
	if e = os.Remove(file); e != nil {
		t.Fatal(e)
	}
	s.Reindex()
	eventually(t, func() bool { return len(s.Search("new-document", "file", 2).Results) == 0 })
}
