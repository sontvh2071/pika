package catalog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"pika/internal/config"
	"testing"
	"time"
)

func TestSearch(t *testing.T) {
	if got := Normalize("Điều Khiển ĐỆM"); got != "dieu khien dem" {
		t.Fatal(got)
	}
	items := []Candidate{Prepare(Candidate{ID: "a", Kind: "app", Name: "Firefox", Aliases: []string{"browser"}}), Prepare(Candidate{ID: "b", Kind: "file", Name: "Firefox settings", Subtitle: "~/Documents"}), Prepare(Candidate{ID: "c", Kind: "command", Name: "Open project", Aliases: []string{"pika"}}), Prepare(Candidate{ID: "d", Kind: "file", Name: "Điều Khiển.txt"})}
	usage := map[string]Usage{"b": {Count: 1000000, LastUsed: time.Now().Unix()}}
	// Usage may win within a fuzzy tier, but must never beat an exact match.
	for _, tc := range []struct{ q, kind, id string }{{"firefox", "all", "a"}, {"frfx", "all", "b"}, {"browser", "all", "a"}, {"> pika", "all", "c"}, {"dieu khien", "all", "d"}, {"firefox", "file", "b"}} {
		got := Search(items, usage, nil, tc.q, tc.kind, 10, time.Now())
		if len(got) == 0 || got[0].ID != tc.id {
			t.Fatalf("%s: %+v", tc.q, got)
		}
	}
	if got := Search(items, usage, []string{"a"}, "unrelated", "all", 10, time.Now()); len(got) != 0 {
		t.Fatal(got)
	}
}
func put(t *testing.T, path, s string) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(path, []byte(s), 0600); e != nil {
		t.Fatal(e)
	}
}
func TestDesktopPrecedence(t *testing.T) {
	dir := t.TempDir()
	user, system := filepath.Join(dir, "user"), filepath.Join(dir, "system")
	entry := "[Desktop Entry]\nType=Application\nName=Firefox\nExec=firefox %u\n"
	put(t, filepath.Join(user, "firefox.desktop"), entry+"Hidden=true\n")
	put(t, filepath.Join(system, "firefox.desktop"), entry)
	put(t, filepath.Join(system, "nested", "terminal.desktop"), "[Desktop Entry]\nType=Application\nName=Terminal\nOnlyShowIn=X-Cinnamon;\n[Desktop Action Fake]\nName=Wrong\n")
	t.Setenv("XDG_CURRENT_DESKTOP", "X-Cinnamon")
	items, _, warn := CollectApps(context.Background(), []string{user, system})
	if len(warn) != 0 || len(items) != 1 || items[0].ID != "app:nested-terminal.desktop" || items[0].Name != "Terminal" {
		t.Fatalf("%+v %v", items, warn)
	}
}
func TestFiles(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "project", "Điều khiển.txt"), "hi")
	put(t, filepath.Join(root, "project", "node_modules", "noise.txt"), "ignored")
	put(t, filepath.Join(root, ".secret"), "hidden")
	put(t, filepath.Join(root, "project", "pending.tmp"), "temporary")
	_ = os.Symlink(root, filepath.Join(root, "project", "loop"))
	c := config.Defaults().Index
	c.Roots = []string{root, filepath.Join(root, "project")}
	items, _, warn := CollectFiles(context.Background(), c)
	if len(warn) != 0 {
		t.Fatal(warn)
	}
	seen := map[string]bool{}
	for _, i := range items {
		if seen[i.ID] {
			t.Fatal("duplicate", i.ID)
		}
		seen[i.ID] = true
		if i.Name == "noise.txt" || i.Name == ".secret" || i.Name == "pending.tmp" {
			t.Fatal(i)
		}
	}
	if !seen["file:"+filepath.Join(root, "project", "Điều khiển.txt")] {
		t.Fatal("file missing")
	}
}
func benchmarkSearch(b *testing.B, n int) {
	items := make([]Candidate, n)
	for i := range items {
		items[i] = Prepare(Candidate{ID: fmt.Sprint(i), Name: fmt.Sprintf("project document %d", i), Kind: "file", Subtitle: "~/Documents/work"})
	}
	u := map[string]Usage{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Search(items, u, nil, "prj", "all", 10, time.Unix(0, 0))
	}
}
func BenchmarkSearch10K(b *testing.B) { benchmarkSearch(b, 10000) }
func BenchmarkSearch50K(b *testing.B) { benchmarkSearch(b, 50000) }

func TestHomeScopeAndFolderSymlink(t *testing.T) {
	home := t.TempDir()
	outside := t.TempDir()
	put(t, filepath.Join(outside, "outside.txt"), "")
	for _, name := range []string{"Downloads/archive.zip", "Desktop/note.txt", "Pictures/photo.jpg", "workspace/project/main.go"} {
		put(t, filepath.Join(home, name), "")
	}
	if err := os.Symlink(outside, filepath.Join(home, "workspace", "shortcut")); err != nil {
		t.Fatal(err)
	}
	cfg := config.Defaults().Index
	cfg.Roots = []string{home}
	items, _, warnings := CollectFiles(context.Background(), cfg)
	if len(warnings) > 0 {
		t.Fatal(warnings)
	}
	seen := map[string]Candidate{}
	for _, item := range items {
		seen[item.Path] = item
		if item.Name == "outside.txt" {
			t.Fatal("followed external symlink")
		}
	}
	for _, name := range []string{"Downloads/archive.zip", "Desktop/note.txt", "Pictures/photo.jpg", "workspace/project/main.go"} {
		if _, ok := seen[filepath.Join(home, name)]; !ok {
			t.Fatal("missing", name)
		}
	}
	if seen[filepath.Join(home, "workspace", "shortcut")].Kind != "directory" {
		t.Fatal("symlink to folder has wrong kind")
	}
	results := Search(items, nil, nil, "", "file", 20, time.Now())
	if len(results) == 0 {
		t.Fatal("Files filter empty despite indexed files")
	}
}

func TestExcludeSpecificSubtree(t *testing.T) {
	home := t.TempDir()
	excluded := filepath.Join(home, "workspace", "go")
	homeGo := filepath.Join(home, "go")
	for _, rel := range []string{"go/pkg/mod/noise.go", "go/bin/tool", "workspace/go/pkg/mod/noise.go", "workspace/go/project/main.go", "go-tools/keep.go", "workspace/go-tools/keep.go", "Documents/go/keep.txt", "workspace/pika/main.go"} {
		put(t, filepath.Join(home, rel), "")
	}
	cfg := config.Defaults().Index
	cfg.Roots = []string{home, excluded, homeGo} // An explicitly listed root cannot override exclusion.
	cfg.ExcludePaths = []string{excluded, homeGo}
	items, dirs, warnings := CollectFiles(context.Background(), cfg)
	if len(warnings) > 0 {
		t.Fatal(warnings)
	}
	blocked := ExcludedPaths(cfg.ExcludePaths)
	seen := map[string]bool{}
	for _, item := range items {
		if blocked.Contains(item.Path) {
			t.Fatal("excluded candidate", item.Path)
		}
		seen[item.Path] = true
	}
	for _, dir := range dirs {
		if blocked.Contains(dir) {
			t.Fatal("excluded directory would be watched", dir)
		}
	}
	for _, rel := range []string{"go-tools/keep.go", "workspace/go-tools/keep.go", "Documents/go/keep.txt", "workspace/pika/main.go"} {
		if !seen[filepath.Join(home, rel)] {
			t.Fatal("overbroad exclusion", rel)
		}
	}
}
