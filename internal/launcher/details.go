package launcher

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"pika/internal/config"
)

// InWorkspace matches whole path components, including the workspace root itself.
// The lexical path governs routing: a symlink selected under workspace opens in
// the editor, without following directory symlinks during indexing.
func InWorkspace(root, path string) bool {
	if root == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(config.Expand(root)), filepath.Clean(path))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}
func FileCommand(cfg config.Open, path string) *exec.Cmd {
	if InWorkspace(cfg.WorkspaceRoot, path) {
		return exec.Command(config.Expand(cfg.WorkspaceExecutable), "--reuse-window", "--", path)
	}
	return exec.Command("xdg-open", path)
}

type Details struct {
	UsageProvider string `json:"usage_provider,omitempty"`
	Kind          string `json:"kind"`
	Path          string `json:"path"`
	Version       string `json:"version"`
	Opener        string `json:"opener"`
}
type cachedDetails struct {
	Generation uint64
	Value      Details
}

func (s *Service) Details(id string) (Details, error) {
	snap := s.snapshot.Load()
	c, ok := snap.ByID[id]
	if !ok {
		return Details{}, fmt.Errorf("Result is no longer indexed")
	}
	d := Details{Kind: c.Kind, Path: c.Path}
	if supportsCodexUsage(c) {
		d.UsageProvider = "codex"
	}
	switch c.Kind {
	case "file", "directory":
		d.Path = c.Target
		d.Opener = "Default application"
		if c.Kind == "directory" {
			d.Opener = "File manager"
		}
		if InWorkspace(s.Config().Open.WorkspaceRoot, c.Target) {
			d.Opener = "VS Code"
		}
	case "system":
		if c.Target == "sensors" {
			d.Opener = "Refresh readings"
		} else if c.Target == "lock" {
			d.Opener = "Lock screen"
		} else {
			d.Opener = "Show options"
		}
	case "command":
		d.Opener = "Run command"
	case "app":
		d.Opener = "Launch application"
		if cached, ok := s.details.Load(id); ok {
			v := cached.(cachedDetails)
			if v.Generation == snap.Version {
				return v.Value, nil
			}
		}
		ctx, cancel := context.WithTimeout(s.ctx, 1800*time.Millisecond)
		defer cancel()
		select {
		case s.metadataSlots <- struct{}{}:
			defer func() { <-s.metadataSlots }()
		case <-ctx.Done():
			return d, nil
		}
		d.Version = c.VersionHint
		if d.Version == "" && c.FlatpakID != "" {
			scope := "--system"
			if strings.Contains(c.Path, "/.local/share/flatpak/") {
				scope = "--user"
			}
			d.Version = commandText(ctx, "flatpak", "info", scope, "--show-version", c.FlatpakID)
		}
		if d.Version == "" {
			d.Version = debVersion(ctx, c.Path)
		}
		// Desktop Entry's standard Version key is the specification version, not
		// the application's version. Never display it or execute arbitrary --version.
		if ctx.Err() == nil {
			s.details.Store(id, cachedDetails{snap.Version, d})
		}
	}
	return d, nil
}
func commandText(ctx context.Context, name string, args ...string) string {
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil || len(out) > 16384 {
		return ""
	}
	return strings.TrimSpace(string(out))
}
func debVersion(ctx context.Context, path string) string {
	if path == "" {
		return ""
	}
	candidates := []string{path}
	if real, err := filepath.EvalSymlinks(path); err == nil && real != path {
		candidates = append(candidates, real)
	}
	for _, p := range candidates {
		owner := commandText(ctx, "dpkg-query", "--search", p)
		for _, line := range strings.Split(owner, "\n") {
			pkg, ownedPath, ok := strings.Cut(line, ": ")
			if !ok || ownedPath != p {
				continue
			}
			// Accept exactly one package, not a diversion or an ambiguous owner list.
			if strings.ContainsAny(pkg, " ,\t\n") {
				continue
			}
			version := commandText(ctx, "dpkg-query", "--show", "--showformat=${Version}", pkg)
			if version != "" {
				return version
			}
		}
	}
	return ""
}
