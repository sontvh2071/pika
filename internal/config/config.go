package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	SchemaVersion int        `toml:"schema_version" json:"schema_version"`
	Window        Window     `toml:"window" json:"window"`
	Appearance    Appearance `toml:"appearance" json:"appearance"`
	Search        Search     `toml:"search" json:"search"`
	Open          Open       `toml:"open" json:"open"`
	Index         Index      `toml:"index" json:"index"`
	Watcher       Watcher    `toml:"watcher" json:"watcher"`
	Commands      []Command  `toml:"commands" json:"commands"`
	Pinned        []string   `toml:"pinned_ids" json:"pinned_ids"`
}
type Window struct {
	Width         int  `toml:"width" json:"width"`
	Height        int  `toml:"height" json:"height"`
	RememberQuery bool `toml:"remember_query" json:"remember_query"`
}
type Appearance struct {
	Theme      string            `toml:"theme" json:"theme"`
	InnerInset int               `toml:"inner_inset" json:"inner_inset"`
	Radius     int               `toml:"radius" json:"radius"`
	FontSize   int               `toml:"font_size" json:"font_size"`
	Colors     map[string]string `toml:"colors" json:"colors"`
}
type Open struct {
	WorkspaceRoot       string `toml:"workspace_root" json:"workspace_root"`
	WorkspaceExecutable string `toml:"workspace_executable" json:"workspace_executable"`
}
type Search struct {
	IncludeCommands bool `toml:"include_commands" json:"include_commands"`
	MaxResults      int  `toml:"max_results" json:"max_results"`
}
type Index struct {
	ExcludePaths  []string `toml:"exclude_paths" json:"exclude_paths"`
	Roots         []string `toml:"roots" json:"roots"`
	ExcludeDirs   []string `toml:"exclude_dirs" json:"exclude_dirs"`
	MaxDepth      int      `toml:"max_depth" json:"max_depth"`
	MaxCandidates int      `toml:"max_candidates" json:"max_candidates"`
	IncludeHidden bool     `toml:"include_hidden" json:"include_hidden"`
}
type Watcher struct {
	Enabled          bool `toml:"enabled" json:"enabled"`
	MaxDirectories   int  `toml:"max_directories" json:"max_directories"`
	ReconcileSeconds int  `toml:"reconcile_seconds" json:"reconcile_seconds"`
}
type Command struct {
	ID         string   `toml:"id" json:"id"`
	Name       string   `toml:"name" json:"name"`
	Aliases    []string `toml:"aliases" json:"aliases"`
	Executable string   `toml:"executable" json:"executable"`
	Args       []string `toml:"args" json:"args"`
	Cwd        string   `toml:"cwd" json:"cwd"`
}

const Example = `# Pika — Tokyo Night. Reload with Ctrl+Shift+, or pika reload-config.
schema_version = 1
pinned_ids = []

[window]
width = 800
height = 660
remember_query = false

[appearance]
theme = "tokyo-night" # tokyo-night, catppuccin, rose-pine, gruvbox, dracula, kanagawa, light, custom
inner_inset = 12      # spacing between search and result panels, in px
radius = 20
font_size = 15

# For theme = "custom", override any of these semantic colors.
[appearance.colors]
background = "#1a1b26"
surface = "#24283b"
text = "#c0caf5"
muted = "#9aa5ce"
selection = "#283457"
accent = "#7aa2f7"
border = "#414868"
error = "#f7768e"

[search]
max_results = 10
include_commands = true

[open]
workspace_root = "~/workspace"
workspace_executable = "code"

[index]
# Empty by default. Add only directories you want searchable.
roots = [] # e.g. ["~/Documents", "~/Downloads", "~/workspace/me"]
exclude_paths = [] # absolute paths or ~/paths; excludes that subtree only
exclude_dirs = [".git", "node_modules", ".cache", "vendor", "dist", "build", ".venv"]
max_depth = 8
max_candidates = 50000
include_hidden = false

[watcher]
enabled = true
max_directories = 4096
reconcile_seconds = 900

# Uncomment and adjust to create your own project shortcut.
# [[commands]]
# id = "open-pika"
# name = "Open Pika project"
# aliases = ["pika", "project pika"]
# executable = "code"
# args = ["/home/lilmint/workspace/me/pika"]
# cwd = "/home/lilmint/workspace/me/pika"
`

func Defaults() Config { var c Config; _, _ = toml.Decode(Example, &c); return c }
func HomePath(env, fallback string) string {
	if p := os.Getenv(env); filepath.IsAbs(p) {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, fallback)
}
func Path() string {
	return filepath.Join(HomePath("XDG_CONFIG_HOME", ".config"), "pika", "config.toml")
}
func DataPath() string {
	return filepath.Join(HomePath("XDG_DATA_HOME", ".local/share"), "pika", "state.db")
}
func Expand(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		h, _ := os.UserHomeDir()
		return filepath.Join(h, strings.TrimPrefix(p, "~/"))
	}
	return p
}

func Load(path string, create bool) (Config, error) {
	c := Defaults()
	if create {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				return c, err
			}
			f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if e == nil {
				_, e = f.WriteString(Example)
				closeErr := f.Close()
				if e == nil {
					e = closeErr
				}
			}
			if e != nil && !os.IsExist(e) {
				return c, e
			}
		}
	}
	meta, err := toml.DecodeFile(path, &c)
	if err != nil {
		return c, fmt.Errorf("config: %w", err)
	}
	if unknown := meta.Undecoded(); len(unknown) > 0 {
		return c, fmt.Errorf("unknown config field: %s", unknown[0])
	}
	if err = c.Validate(); err != nil {
		return c, err
	}
	return c, nil
}
func (c Config) Validate() error {
	if c.SchemaVersion != 1 {
		return fmt.Errorf("schema_version must be 1")
	}
	if c.Window.Width < 480 || c.Window.Width > 1200 || c.Window.Height < 380 || c.Window.Height > 900 {
		return fmt.Errorf("window: width 480–1200, height 380–900 required")
	}
	if c.Appearance.InnerInset < 0 || c.Appearance.InnerInset > 32 || c.Appearance.Radius < 0 || c.Appearance.Radius > 40 || c.Appearance.FontSize < 12 || c.Appearance.FontSize > 24 {
		return fmt.Errorf("appearance: inset 0–32, radius 0–40, font_size 12–24 required")
	}
	if !ValidTheme(c.Appearance.Theme) {
		return fmt.Errorf("unknown theme %q", c.Appearance.Theme)
	}
	color := regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	allowed := map[string]bool{"background": true, "surface": true, "text": true, "muted": true, "selection": true, "accent": true, "border": true, "error": true}
	for k, v := range c.Appearance.Colors {
		if !allowed[k] || !color.MatchString(v) {
			return fmt.Errorf("invalid appearance.colors.%s", k)
		}
	}
	if c.Search.MaxResults < 1 || c.Search.MaxResults > 20 {
		return fmt.Errorf("search.max_results must be 1–20")
	}
	if c.Index.MaxDepth < 1 || c.Index.MaxDepth > 32 || c.Index.MaxCandidates < 100 || c.Index.MaxCandidates > 100000 {
		return fmt.Errorf("index: max_depth 1–32, max_candidates 100–100000 required")
	}
	if c.Watcher.MaxDirectories < 1 || c.Watcher.MaxDirectories > 16384 || c.Watcher.ReconcileSeconds < 30 {
		return fmt.Errorf("watcher: max_directories 1–16384 and reconcile_seconds >=30 required")
	}
	for _, p := range c.Index.Roots {
		if !filepath.IsAbs(Expand(p)) {
			return fmt.Errorf("index root must be absolute or start with ~/: %s", p)
		}
	}
	for _, p := range c.Index.ExcludePaths {
		if !filepath.IsAbs(Expand(p)) {
			return fmt.Errorf("index.exclude_paths must be absolute or start with ~/: %s", p)
		}
	}
	if c.Open.WorkspaceRoot != "" && !filepath.IsAbs(Expand(c.Open.WorkspaceRoot)) {
		return fmt.Errorf("open.workspace_root must be absolute or start with ~/")
	}
	if c.Open.WorkspaceRoot != "" && strings.TrimSpace(c.Open.WorkspaceExecutable) == "" {
		return fmt.Errorf("open.workspace_executable is required")
	}
	ids := map[string]bool{}
	for _, cmd := range c.Commands {
		if cmd.ID == "" || cmd.Name == "" || cmd.Executable == "" || ids[cmd.ID] || strings.ContainsAny(cmd.ID, " \t\n") {
			return fmt.Errorf("command needs unique id, name and executable: %q", cmd.ID)
		}
		ids[cmd.ID] = true
		if cmd.Cwd != "" && !filepath.IsAbs(Expand(cmd.Cwd)) {
			return fmt.Errorf("command cwd must be absolute: %s", cmd.ID)
		}
	}
	return nil
}
