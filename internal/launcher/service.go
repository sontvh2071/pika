package launcher

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"pika/internal/catalog"
	"pika/internal/codexusage"
	"pika/internal/config"
	"pika/internal/desktop"
	"pika/internal/processmonitor"
	"pika/internal/sensors"
	"pika/internal/storage"
)

type Snapshot struct {
	Items   []catalog.Candidate
	ByID    map[string]catalog.Candidate
	Version uint64
}
type Status struct {
	Apps       int      `json:"apps"`
	Files      int      `json:"files"`
	Commands   int      `json:"commands"`
	System     int      `json:"system"`
	Indexing   bool     `json:"indexing"`
	Version    uint64   `json:"version"`
	LastIndex  string   `json:"last_index"`
	DurationMS int64    `json:"duration_ms"`
	Watches    int      `json:"watches"`
	Warnings   []string `json:"warnings"`
	StateError string   `json:"state_error"`
}
type Response struct {
	RequestID  uint64           `json:"request_id"`
	Results    []catalog.Result `json:"results"`
	Version    uint64           `json:"version"`
	DurationMS float64          `json:"duration_ms"`
}
type Service struct {
	themeMu         sync.Mutex
	mu              sync.RWMutex
	cfg             config.Config
	configPath      string
	status          Status
	snapshot        atomic.Pointer[Snapshot]
	usage           atomic.Pointer[map[string]catalog.Usage]
	usageMu         sync.Mutex
	usageVersion    atomic.Uint64
	store           *storage.Store
	refresh         chan struct{}
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	Notify          func(string)
	icons           sync.Map
	details         sync.Map
	metadataSlots   chan struct{}
	codexQuota      *codexusage.Client
	sensorReadings  *sensors.Client
	processReadings *processmonitor.Client
}

func New(cfg config.Config, path, dataPath string) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{cfg: cfg, configPath: path, refresh: make(chan struct{}, 1), ctx: ctx, cancel: cancel, metadataSlots: make(chan struct{}, 2), codexQuota: codexusage.New(), sensorReadings: sensors.New(), processReadings: processmonitor.New()}
	s.snapshot.Store(&Snapshot{Items: []catalog.Candidate{}, ByID: map[string]catalog.Candidate{}})
	usage := map[string]catalog.Usage{}
	store, err := storage.Open(dataPath)
	if err != nil {
		s.status.StateError = err.Error()
	} else {
		s.store = store
		usage, err = store.Load()
		if err != nil {
			s.status.StateError = err.Error()
		}
	}
	s.usage.Store(&usage)
	return s
}
func (s *Service) Start() { s.wg.Add(2); go s.run(); go s.persist() }
func (s *Service) Close() {
	s.cancel()
	s.wg.Wait()
	if s.store != nil {
		_ = s.store.Close()
	}
}
func (s *Service) Config() config.Config { s.mu.RLock(); defer s.mu.RUnlock(); return s.cfg }
func (s *Service) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v := s.status
	v.Warnings = append([]string{}, v.Warnings...)
	return v
}
func (s *Service) emit(event string) {
	if s.Notify != nil {
		s.Notify(event)
	}
}
func (s *Service) Reindex() {
	select {
	case s.refresh <- struct{}{}:
	default:
	}
}
func (s *Service) Reload() error {
	s.themeMu.Lock()
	defer s.themeMu.Unlock()
	c, err := config.Load(s.configPath, false)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.cfg = c
	s.mu.Unlock()
	s.Reindex()
	s.emit("config")
	return nil
}
func (s *Service) Search(query, kind string, id uint64) Response {
	if len([]rune(query)) > 256 {
		query = string([]rune(query)[:256])
	}
	start := time.Now()
	snap := s.snapshot.Load()
	cfg := s.Config()
	results := catalog.Search(snap.Items, *s.usage.Load(), cfg.Pinned, query, kind, cfg.Search.MaxResults, time.Now())
	return Response{id, results, snap.Version, float64(time.Since(start).Microseconds()) / 1000}
}

func (s *Service) SetTheme(name string) error {
	s.themeMu.Lock()
	defer s.themeMu.Unlock()
	if err := config.SetTheme(s.configPath, name); err != nil {
		return err
	}
	s.mu.Lock()
	s.cfg.Appearance.Theme = name
	s.mu.Unlock()
	s.emit("config")
	return nil
}
func (s *Service) record(id string) {
	s.usageMu.Lock()
	old := *s.usage.Load()
	next := make(map[string]catalog.Usage, len(old)+1)
	for k, v := range old {
		next[k] = v
	}
	u := next[id]
	u.Count++
	u.LastUsed = time.Now().Unix()
	next[id] = u
	s.usage.Store(&next)
	s.usageVersion.Add(1)
	s.usageMu.Unlock()
}
func (s *Service) persist() {
	defer s.wg.Done()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	var saved uint64
	flush := func() {
		v := s.usageVersion.Load()
		if v == saved || s.store == nil {
			return
		}
		err := s.store.Save(*s.usage.Load())
		s.mu.Lock()
		if err != nil {
			s.status.StateError = err.Error()
		} else {
			s.status.StateError = ""
			saved = v
		}
		s.mu.Unlock()
	}
	for {
		select {
		case <-s.ctx.Done():
			flush()
			return
		case <-tick.C:
			flush()
		}
	}
}
func (s *Service) publish(items []catalog.Candidate) {
	old := s.snapshot.Load()
	snap := &Snapshot{Items: items, ByID: make(map[string]catalog.Candidate, len(items)), Version: old.Version + 1}
	for _, c := range items {
		snap.ByID[c.ID] = c
	}
	s.snapshot.Store(snap)
}
func (s *Service) run() {
	defer s.wg.Done()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("watcher unavailable", "error", err)
	}
	if watcher != nil {
		defer watcher.Close()
	}
	dirty := make(chan struct{}, 1)
	watchErrors := make(chan string, 1)
	if watcher != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for {
				select {
				case <-s.ctx.Done():
					return
				case e, ok := <-watcher.Events:
					if !ok {
						return
					}
					if e.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 || strings.HasSuffix(e.Name, ".desktop") {
						select {
						case dirty <- struct{}{}:
						default:
						}
					}
				case e, ok := <-watcher.Errors:
					if !ok {
						return
					}
					select {
					case watchErrors <- e.Error():
					default:
					}
					select {
					case dirty <- struct{}{}:
					default:
					}
				}
			}
		}()
	}
	watches := map[string]bool{}
	var lastScan time.Time
	var nextRefresh time.Time
	scan := func() {
		start := time.Now()
		cfg := s.Config()
		s.mu.Lock()
		s.status.Indexing = true
		s.mu.Unlock()
		s.emit("index")
		apps, appDirs, warnings := catalog.CollectApps(s.ctx, catalog.AppRoots())
		commands := []catalog.Candidate{}
		if cfg.Search.IncludeCommands {
			commands = catalog.CollectCommands(cfg.Commands)
		}
		system := catalog.CollectSystemActions()
		base := append(apps, commands...)
		base = append(base, system...)
		if s.snapshot.Load().Version == 0 {
			s.publish(append([]catalog.Candidate{}, base...))
			s.emit("index")
		}
		files, fileDirs, fileWarnings := catalog.CollectFiles(s.ctx, cfg.Index)
		warnings = append(warnings, fileWarnings...)
		if s.ctx.Err() != nil {
			return
		}
		// Preserve previous candidates only for configured roots temporarily unavailable.
		blocked := catalog.ExcludedPaths(cfg.Index.ExcludePaths)
		for _, root := range cfg.Index.Roots {
			p := filepath.Clean(config.Expand(root))
			if _, e := os.Stat(p); e != nil {
				for _, c := range s.snapshot.Load().Items {
					if !blocked.Contains(c.Target) && (c.Kind == "file" || c.Kind == "directory") && (c.Target == p || strings.HasPrefix(c.Target, p+"/")) {
						files = append(files, c)
					}
				}
			}
		}
		base = append(base, files...)
		s.publish(base)
		desired := map[string]bool{}
		// inotify removes watches when a directory is deleted. Reconcile the real
		// watch list so recreating the same path registers a fresh watch.
		watches = map[string]bool{}
		if watcher != nil {
			for _, dir := range watcher.WatchList() {
				watches[dir] = true
			}
		}
		if cfg.Watcher.Enabled && watcher != nil {
			for _, dir := range append(appDirs, fileDirs...) {
				if len(desired) >= cfg.Watcher.MaxDirectories {
					warnings = append(warnings, "Watch limit reached; remaining directories update on periodic/manual reindex")
					break
				}
				desired[dir] = true
			}
		}
		if cfg.Watcher.Enabled && watcher == nil {
			warnings = append(warnings, "Watcher unavailable; periodic/manual reindex only")
		}
		for dir := range watches {
			if !desired[dir] {
				_ = watcher.Remove(dir)
				delete(watches, dir)
			}
		}
		for dir := range desired {
			if !watches[dir] {
				if e := watcher.Add(dir); e != nil {
					warnings = append(warnings, "Watch failed: "+dir+": "+e.Error())
				} else {
					watches[dir] = true
				}
			}
		}
		if len(warnings) > 20 {
			warnings = append(warnings[:20], "Additional warnings omitted")
		}
		s.mu.Lock()
		s.status.Apps = len(apps)
		s.status.Files = len(files)
		s.status.Commands = len(commands)
		s.status.System = len(system)
		s.status.Indexing = false
		s.status.Version = s.snapshot.Load().Version
		s.status.LastIndex = time.Now().Format(time.RFC3339)
		s.status.DurationMS = time.Since(start).Milliseconds()
		s.status.Watches = len(watches)
		s.status.Warnings = warnings
		s.mu.Unlock()
		s.emit("index")
		lastScan = time.Now()
	}
	scan()
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.refresh:
			scan()
			nextRefresh = time.Time{}
		case <-dirty:
			if nextRefresh.IsZero() {
				nextRefresh = time.Now().Add(time.Second)
			}
			min := lastScan.Add(3 * time.Second)
			if nextRefresh.Before(min) {
				nextRefresh = min
			}
		case msg := <-watchErrors:
			s.mu.Lock()
			s.status.Warnings = append(s.status.Warnings, "Watch degraded: "+msg)
			s.mu.Unlock()
			s.emit("index")
		case <-tick.C:
			cfg := s.Config()
			if (!nextRefresh.IsZero() && !time.Now().Before(nextRefresh)) || time.Since(lastScan) >= time.Duration(cfg.Watcher.ReconcileSeconds)*time.Second {
				scan()
				nextRefresh = time.Time{}
			}
		}
	}
}

func (s *Service) Execute(id string) error {
	c, ok := s.snapshot.Load().ByID[id]
	if !ok {
		return fmt.Errorf("This result is no longer available. Search again.")
	}
	var cmd *exec.Cmd
	switch c.Kind {
	case "app":
		if err := desktop.Launch(c.Path); err != nil {
			return err
		}
		s.record(id)
		return nil
	case "system":
		var err error
		cmd, err = SystemCommand(c.Target)
		if err != nil {
			return err
		}
	case "file", "directory":
		if _, err := os.Stat(c.Target); err != nil {
			s.Reindex()
			return fmt.Errorf("File is no longer available: %w", err)
		}
		cmd = FileCommand(s.Config().Open, c.Target)
	case "command":
		for _, cc := range s.Config().Commands {
			if cc.ID == c.Target {
				cmd = exec.Command(config.Expand(cc.Executable), cc.Args...)
				if cc.Cwd != "" {
					cmd.Dir = config.Expand(cc.Cwd)
				}
				break
			}
		}
	}
	if cmd == nil {
		return fmt.Errorf("Command is no longer configured")
	}
	return s.launch(cmd, id)
}
func (s *Service) OpenConfig() error { return s.launch(exec.Command("xdg-open", s.configPath), "") }
func (s *Service) launch(cmd *exec.Cmd, id string) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("Launch failed: %w", err)
		}
	case <-time.After(120 * time.Millisecond):
		go func() {
			if err := <-done; err != nil {
				slog.Warn("launched process exited", "error", err)
				s.mu.Lock()
				s.status.Warnings = append(s.status.Warnings, "A launched command exited with an error; see terminal logs")
				s.mu.Unlock()
				s.emit("index")
			}
		}()
	}
	if id != "" {
		s.record(id)
	}
	return nil
}
func (s *Service) Icon(id string) string {
	c, ok := s.snapshot.Load().ByID[id]
	if !ok || c.Icon == "" {
		return ""
	}
	if v, ok := s.icons.Load(c.Icon); ok {
		return v.(string)
	}
	paths := []string{}
	if filepath.IsAbs(c.Icon) {
		paths = append(paths, c.Icon)
	} else if !strings.Contains(c.Icon, "/") && !strings.ContainsAny(c.Icon, "*?[") {
		for _, ext := range []string{".png", ".svg"} {
			for _, pattern := range []string{"/usr/share/icons/hicolor/*/apps/", "/usr/share/icons/Mint-Y/apps/*/", "/usr/share/icons/Adwaita/*/apps/", "/usr/share/pixmaps/"} {
				matches, _ := filepath.Glob(pattern + c.Icon + ext)
				paths = append(paths, matches...)
			}
		}
	}
	data := ""
	for _, path := range paths {
		info, e := os.Stat(path)
		if e != nil || info.Size() > 256*1024 {
			continue
		}
		b, e := os.ReadFile(path)
		if e != nil {
			continue
		}
		mime := "image/png"
		if strings.HasSuffix(path, ".svg") {
			mime = "image/svg+xml"
		} else if !strings.HasSuffix(path, ".png") {
			continue
		}
		data = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
		break
	}
	s.icons.Store(c.Icon, data)
	return data
}
