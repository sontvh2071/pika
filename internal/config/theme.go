package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

func ValidTheme(name string) bool {
	switch name {
	case "tokyo-night", "catppuccin", "rose-pine", "gruvbox", "dracula", "kanagawa", "light", "custom":
		return true
	}
	return false
}

// SetTheme edits only appearance.theme, preserving the user's comments and
// commands. Invalid configs are left untouched for the user to correct.
func SetTheme(path, name string) error {
	if !ValidTheme(name) {
		return fmt.Errorf("unknown theme %q", name)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	c := Defaults()
	meta, err := toml.Decode(string(original), &c)
	if err != nil {
		return err
	}
	if len(meta.Undecoded()) > 0 {
		return fmt.Errorf("fix unknown config fields before changing theme")
	}
	if err = c.Validate(); err != nil {
		return err
	}
	lines := strings.Split(string(original), "\n")
	section, found, insertAt := false, false, -1
	key := regexp.MustCompile(`^\s*theme\s*=`)
	for i, line := range lines {
		clean := strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if strings.HasPrefix(clean, "[") {
			section = clean == "[appearance]"
			if section {
				insertAt = i + 1
			}
		}
		if section && key.MatchString(line) {
			comment := ""
			if at := strings.Index(line, "#"); at >= 0 {
				comment = " " + line[at:]
			}
			lines[i] = fmt.Sprintf("theme = %q%s", name, comment)
			found = true
			break
		}
	}
	if !found {
		line := fmt.Sprintf("theme = %q", name)
		if insertAt < 0 {
			lines = append(lines, "[appearance]", line)
		} else {
			lines = append(lines[:insertAt], append([]string{line}, lines[insertAt:]...)...)
		}
	}
	updated := []byte(strings.Join(lines, "\n"))
	c = Defaults()
	if _, err = toml.Decode(string(updated), &c); err != nil {
		return err
	}
	if err = c.Validate(); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".pika-theme-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(updated); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(original, current) {
		return fmt.Errorf("config changed in your editor; retry theme selection")
	}
	return os.Rename(f.Name(), path)
}
