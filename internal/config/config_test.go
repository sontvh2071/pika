package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadValidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	c, e := Load(path, true)
	if e != nil {
		t.Fatal(e)
	}
	if c.Appearance.InnerInset != 12 || c.Appearance.Theme != "tokyo-night" {
		t.Fatal(c)
	}
	for _, body := range []string{strings.Replace(Example, "inner_inset = 12", "inner_inset = 100", 1), Example + "\nunknown = 1\n", strings.Replace(Example, "schema_version = 1", "schema_version = 99", 1)} {
		if e = os.WriteFile(path, []byte(body), 0600); e != nil {
			t.Fatal(e)
		}
		if _, e = Load(path, false); e == nil {
			t.Fatal("accepted invalid config")
		}
	}
}

func TestExcludePathsValidation(t *testing.T) {
	cfg := Defaults()
	cfg.Index.ExcludePaths = []string{"workspace/go"}
	if cfg.Validate() == nil {
		t.Fatal("relative exclusion accepted")
	}
	cfg.Index.ExcludePaths = []string{"~/workspace/go", "/tmp/excluded"}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}
