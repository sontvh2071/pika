package catalog

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"pika/internal/config"
)

func AppRoots() []string {
	roots := []string{filepath.Join(config.HomePath("XDG_DATA_HOME", ".local/share"), "applications")}
	dirs := os.Getenv("XDG_DATA_DIRS")
	if dirs == "" {
		dirs = "/usr/local/share:/usr/share"
	}
	for _, dir := range filepath.SplitList(dirs) {
		if filepath.IsAbs(dir) {
			roots = append(roots, filepath.Join(dir, "applications"))
		}
	}
	home, _ := os.UserHomeDir()
	roots = append(roots, filepath.Join(home, ".local/share/flatpak/exports/share/applications"), "/var/lib/flatpak/exports/share/applications")
	seen := map[string]bool{}
	out := []string{}
	for _, r := range roots {
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	return out
}
func localeNames() []string {
	l := os.Getenv("LC_ALL")
	if l == "" {
		l = os.Getenv("LC_MESSAGES")
	}
	if l == "" {
		l = os.Getenv("LANG")
	}
	l = strings.Split(l, ".")[0]
	parts := strings.SplitN(l, "@", 2)
	base := parts[0]
	lang := strings.Split(base, "_")[0]
	out := []string{l, base}
	if len(parts) == 2 {
		out = append(out, lang+"@"+parts[1])
	}
	return append(out, lang)
}
func unescape(s string) string {
	return strings.NewReplacer(`\s`, " ", `\n`, "\n", `\t`, "\t", `\r`, "\r", `\\`, `\`).Replace(s)
}
func DesktopEntry(path, id, desktop string) (Candidate, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return Candidate{}, false, err
	}
	defer f.Close()
	vals := map[string]string{}
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 4096), 1024*1024)
	active := false
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if strings.HasPrefix(line, "[") {
			active = line == "[Desktop Entry]"
			continue
		}
		if !active || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok {
			vals[strings.TrimSpace(k)] = unescape(strings.TrimSpace(v))
		}
	}
	if err = scan.Err(); err != nil {
		return Candidate{}, false, err
	}
	if vals["Type"] != "Application" || vals["Hidden"] == "true" || vals["NoDisplay"] == "true" {
		return Candidate{}, false, nil
	}
	intersects := func(list string) bool {
		for _, d := range strings.Split(desktop, ":") {
			for _, v := range strings.Split(list, ";") {
				if v != "" && v == d {
					return true
				}
			}
		}
		return false
	}
	if vals["OnlyShowIn"] != "" && !intersects(vals["OnlyShowIn"]) {
		return Candidate{}, false, nil
	}
	if intersects(vals["NotShowIn"]) {
		return Candidate{}, false, nil
	}
	if t := vals["TryExec"]; t != "" {
		if _, e := exec.LookPath(t); e != nil {
			return Candidate{}, false, nil
		}
	}
	localized := func(k string) string {
		for _, l := range localeNames() {
			if s := vals[k+"["+l+"]"]; s != "" {
				return s
			}
		}
		return vals[k]
	}
	name := localized("Name")
	if name == "" {
		return Candidate{}, false, nil
	}
	sub := localized("GenericName")
	if sub == "" {
		sub = "Application"
	}
	aliases := []string{vals["Name"], strings.TrimSuffix(id, ".desktop"), localized("GenericName")}
	aliases = append(aliases, strings.Split(localized("Keywords"), ";")...)
	return Prepare(Candidate{ID: "app:" + id, Kind: "app", Name: name, Subtitle: sub, Target: id, Path: path, VersionHint: vals["X-AppImage-Version"], FlatpakID: vals["X-Flatpak"], Icon: vals["Icon"], Aliases: aliases}), true, nil
}
func CollectApps(ctx context.Context, roots []string) ([]Candidate, []string, []string) {
	items := []Candidate{}
	dirs := []string{}
	warnings := []string{}
	seen := map[string]bool{}
	for _, root := range roots {
		if _, e := os.Stat(root); os.IsNotExist(e) {
			continue
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				return err
			}
			if d.IsDir() {
				dirs = append(dirs, path)
				return nil
			}
			if !strings.HasSuffix(path, ".desktop") {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			id := strings.ReplaceAll(rel, string(filepath.Separator), "-")
			if seen[id] {
				return nil
			}
			seen[id] = true // Hidden user entries mask system entries too.
			c, visible, e := DesktopEntry(path, id, os.Getenv("XDG_CURRENT_DESKTOP"))
			if e != nil {
				warnings = append(warnings, fmt.Sprintf("desktop %s: %v", id, e))
			} else if visible {
				items = append(items, c)
			}
			return nil
		})
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("applications %s: %v", root, err))
		}
	}
	return items, dirs, warnings
}

func CollectFiles(ctx context.Context, cfg config.Index) ([]Candidate, []string, []string) {
	out := []Candidate{}
	dirs := []string{}
	warnings := []string{}
	seen := map[string]bool{}
	seenDir := map[string]bool{}
	blocked := ExcludedPaths(cfg.ExcludePaths)
	excluded := map[string]bool{}
	for _, s := range cfg.ExcludeDirs {
		excluded[s] = true
	}
	roots := append([]string{}, cfg.Roots...)
	sort.Strings(roots)
	for _, raw := range roots {
		root := filepath.Clean(config.Expand(raw))
		if real, e := filepath.EvalSymlinks(root); e == nil {
			root = real
		}
		if blocked.Contains(root) {
			continue
		}
		info, e := os.Stat(root)
		if e != nil || !info.IsDir() {
			warnings = append(warnings, "Unavailable directory: "+raw)
			continue
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("Cannot read %s: %v", path, err))
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if blocked.Contains(path) {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if len(out) >= cfg.MaxCandidates {
				return fs.SkipAll
			}
			rel, _ := filepath.Rel(root, path)
			depth := 0
			if rel != "." {
				depth = len(strings.Split(rel, string(filepath.Separator)))
			}
			if path != root && ((!cfg.IncludeHidden && strings.HasPrefix(d.Name(), ".")) || excluded[d.Name()] || depth > cfg.MaxDepth) {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				if seenDir[path] {
					return fs.SkipDir
				}
				seenDir[path] = true
				dirs = append(dirs, path)
			}
			if seen[path] {
				return nil
			}
			seen[path] = true
			for _, ext := range []string{".swp", ".tmp", ".crdownload", "~"} {
				if strings.HasSuffix(d.Name(), ext) && !d.IsDir() {
					return nil
				}
			}
			if !d.IsDir() && d.Type()&os.ModeSymlink == 0 && !d.Type().IsRegular() {
				return nil
			}
			kind := "file"
			if d.IsDir() {
				kind = "directory"
			} else if d.Type()&os.ModeSymlink != 0 {
				if info, err := os.Stat(path); err == nil && info.IsDir() {
					kind = "directory"
				}
			}
			out = append(out, Prepare(Candidate{ID: kind + ":" + path, Kind: kind, Name: d.Name(), Subtitle: ShortPath(filepath.Dir(path)), Target: path, Path: path}))
			return nil
		})
		if err != nil {
			warnings = append(warnings, err.Error())
		}
		if len(out) >= cfg.MaxCandidates {
			warnings = append(warnings, fmt.Sprintf("File index limited to %d items; narrow roots or exclusions", cfg.MaxCandidates))
			break
		}
	}
	return out, dirs, warnings
}
func ShortPath(p string) string {
	h, _ := os.UserHomeDir()
	if p == h {
		return "~"
	}
	if strings.HasPrefix(p, h+"/") {
		return "~" + strings.TrimPrefix(p, h)
	}
	return p
}
func CollectCommands(commands []config.Command) []Candidate {
	out := []Candidate{}
	for _, c := range commands {
		out = append(out, Prepare(Candidate{ID: "command:" + c.ID, Kind: "command", Name: c.Name, Subtitle: c.Executable + " " + strings.Join(c.Args, " "), Target: c.ID, Aliases: c.Aliases}))
	}
	return out
}

// Path exclusions are component-aware; excluding workspace/go keeps go-tools
// and other directories named go elsewhere. Resolve root aliases once per scan.
type PathExclusions []string

func ExcludedPaths(paths []string) PathExclusions {
	out := PathExclusions{}
	for _, raw := range paths {
		p := filepath.Clean(config.Expand(raw))
		out = append(out, p)
		if real, err := filepath.EvalSymlinks(p); err == nil && real != p {
			out = append(out, real)
		}
	}
	return out
}
func (paths PathExclusions) Contains(path string) bool {
	for _, p := range paths {
		if path == p || strings.HasPrefix(path, strings.TrimRight(p, string(filepath.Separator))+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
