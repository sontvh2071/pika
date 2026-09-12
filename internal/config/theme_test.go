package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetThemePreservesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	original := Example + "\n[[commands]]\nid = \"work\"\nname = \"Work #1\"\nexecutable = \"echo\"\nargs = [\"theme = untouched\"]\n"
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"catppuccin", "rose-pine", "gruvbox", "dracula", "kanagawa", "light", "custom", "tokyo-night"} {
		if err := SetTheme(path, name); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(path, false)
		if err != nil || cfg.Appearance.Theme != name {
			t.Fatalf("theme=%s config=%+v err=%v", name, cfg, err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		want := strings.Replace(original, `theme = "tokyo-night" #`, `theme = "`+name+`" #`, 1)
		if string(got) != want {
			t.Fatal("unrelated config content changed")
		}
	}
}

func TestSetThemeMissingKey(t *testing.T) {
	for _, body := range []string{"", "[window]\nwidth = 720\n", "[appearance] # custom look\nradius = 18\n", "[appearance.colors]\naccent = \"#abcdef\"\n"} {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if err := SetTheme(path, "kanagawa"); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(path, false)
		if err != nil || cfg.Appearance.Theme != "kanagawa" {
			t.Fatalf("%+v %v", cfg, err)
		}
	}
}

func TestSetThemeRejectsWithoutWriting(t *testing.T) {
	for _, tc := range []struct{ body, name string }{
		{Example, "unknown"}, {"not valid toml", "dracula"},
		{"unexpected = 1", "dracula"}, {"[window]\nwidth = 2", "dracula"},
	} {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte(tc.body), 0600); err != nil {
			t.Fatal(err)
		}
		if err := SetTheme(path, tc.name); err == nil {
			t.Fatal("accepted invalid config/theme")
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != tc.body {
			t.Fatal("invalid config modified")
		}
	}
}
