package main

import (
	"context"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"os/signal"
	"pika/internal/codexusage"
	"pika/internal/config"
	"pika/internal/ipc"
	"pika/internal/launcher"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type App struct {
	ctx                      context.Context
	service                  *launcher.Service
	server                   *ipc.Server
	mu                       sync.Mutex
	ready, visible, quitting bool
	initialShow              bool
	dispatching              atomic.Bool
	frontendFocus            atomic.Uint32
	stopSignals              func()
}
type AppState struct {
	Config     config.Config   `json:"config"`
	Status     launcher.Status `json:"status"`
	ConfigPath string          `json:"config_path"`
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	signals := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	a.stopSignals = func() { signal.Stop(signals); close(done) }
	go func() {
		select {
		case <-signals:
			a.Quit()
		case <-done:
		}
	}()
	a.service.Notify = func(event string) { runtime.EventsEmit(ctx, "pika:"+event) }
	a.service.Start()
	a.server.Serve(a.control)
}
func (a *App) shutdown(ctx context.Context) {
	cancelLauncherFocus()
	if a.stopSignals != nil {
		a.stopSignals()
	}
	a.server.Close()
	a.service.Close()
}
func (a *App) beforeClose(ctx context.Context) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.quitting {
		return false
	}
	a.visible = false
	cancelLauncherFocus()
	runtime.WindowHide(ctx)
	runtime.EventsEmit(ctx, "pika:hidden")
	return true
}
func (a *App) showLocked() {
	runtime.WindowShow(a.ctx)
	runtime.WindowCenter(a.ctx)
	focusLauncher()
	a.visible = true
	runtime.EventsEmit(a.ctx, "pika:shown")
}
func (a *App) hideLocked() {
	cancelLauncherFocus()
	runtime.WindowHide(a.ctx)
	a.visible = false
	runtime.EventsEmit(a.ctx, "pika:hidden")
}
func (a *App) FrontendReady() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ready = true
	if a.initialShow {
		a.showLocked()
		a.initialShow = false
	}
}
func (a *App) Hide() { a.mu.Lock(); defer a.mu.Unlock(); a.hideLocked() }
func (a *App) GetState() AppState {
	return AppState{a.service.Config(), a.service.Status(), config.Path()}
}
func (a *App) Search(query, kind string, requestID uint64) launcher.Response {
	return a.service.Search(query, kind, requestID)
}
func (a *App) Details(id string) (launcher.Details, error) { return a.service.Details(id) }
func (a *App) Icon(id string) string                       { return a.service.Icon(id) }
func (a *App) Execute(id string) error {
	if !a.dispatching.CompareAndSwap(false, true) {
		return nil
	}
	defer a.dispatching.Store(false)
	// Release keyboard ownership and the always-on-top surface before the
	// desktop opens an authentication or session dialog.
	a.Hide()
	if e := a.service.Execute(id); e != nil {
		a.mu.Lock()
		a.showLocked()
		a.mu.Unlock()
		return e
	}
	return nil
}

// Report only focus booleans; never expose or log the search query.
func (a *App) ReportFocus(documentFocused, queryFocused bool) {
	var flags uint32
	if documentFocused {
		flags |= 1
	}
	if queryFocused {
		flags |= 2
	}
	a.frontendFocus.Store(flags)
}
func (a *App) focusState() map[string]bool {
	mapped, active, webview := nativeFocusState()
	flags := a.frontendFocus.Load()
	a.mu.Lock()
	visible := a.visible
	a.mu.Unlock()
	return map[string]bool{"visible": visible, "mapped": mapped, "window_active": active, "webview_focused": webview, "document_focused": flags&1 != 0, "query_focused": flags&2 != 0}
}
func (a *App) CodexUsage(id string, refresh bool) (codexusage.Snapshot, error) {
	return a.service.CodexUsage(id, refresh)
}
func (a *App) OpenConfig() error          { return a.service.OpenConfig() }
func (a *App) SetTheme(name string) error { return a.service.SetTheme(name) }
func (a *App) Reindex()                   { a.service.Reindex() }

// Reserve the full layout even when results are hidden, keeping search anchored.
func (a *App) resizeLocked() {
	c := a.service.Config()
	runtime.WindowSetSize(a.ctx, c.Window.Width, c.Window.Height)
}

// GTK resizing is asynchronous. The frontend calls this after a resize event
// so centering uses the allocated size rather than the previous window size.
func (a *App) CenterWindow(width, height int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	cfg := a.service.Config()
	if width != cfg.Window.Width || height != cfg.Window.Height {
		return
	}
	runtime.WindowCenter(a.ctx)
}
func (a *App) ReloadConfig() error {
	if e := a.service.Reload(); e != nil {
		return e
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.resizeLocked()
	return nil
}
func (a *App) Quit() {
	a.mu.Lock()
	a.quitting = true
	cancelLauncherFocus()
	a.mu.Unlock()
	go func() { time.Sleep(100 * time.Millisecond); runtime.Quit(a.ctx) }()
}
func (a *App) control(command string) ipc.Response {
	a.mu.Lock()
	if !a.ready {
		a.mu.Unlock()
		return ipc.Response{Message: "starting"}
	}
	switch command {
	case "show":
		a.showLocked()
	case "hide":
		a.hideLocked()
	case "toggle":
		if a.visible {
			a.hideLocked()
		} else {
			a.showLocked()
		}
	case "--background":
	default:
		a.mu.Unlock()
		switch command {
		case "reindex":
			a.Reindex()
			return ipc.Response{OK: true, Message: "Reindex queued"}
		case "reload-config":
			if e := a.ReloadConfig(); e != nil {
				return ipc.Response{Message: e.Error()}
			}
			return ipc.Response{OK: true, Message: "Configuration reloaded"}
		case "focus-state":
			return ipc.Response{OK: true, Data: a.focusState()}
		case "stats":
			return ipc.Response{OK: true, Data: a.GetState().Status}
		case "quit":
			a.Quit()
			return ipc.Response{OK: true}
		default:
			return ipc.Response{Message: "Unknown command"}
		}
	}
	a.mu.Unlock()
	return ipc.Response{OK: true}
}
