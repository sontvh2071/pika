//go:build linux

// Package desktop launches desktop entries from the long-lived Pika process.
package desktop

/*
#cgo pkg-config: gio-unix-2.0
#include <gio/gdesktopappinfo.h>
#include <stdlib.h>
extern void pikaDesktopChildStarted(int pid);
static void pika_child_started(GDesktopAppInfo *info, GPid pid, gpointer data) {
    pikaDesktopChildStarted((int)pid);
}
static char *pika_launch_desktop(const char *path) {
    GDesktopAppInfo *info = g_desktop_app_info_new_from_filename(path);
    if (!info) return g_strdup("Desktop entry is missing or invalid");
    GAppLaunchContext *context = g_app_launch_context_new();
    GError *error = NULL;
    // Keep Pika as the parent of pkexec. A short-lived gtk-launch or a
    // double-forked child can leave pkexec parentless before authentication.
    gboolean ok = g_desktop_app_info_launch_uris_as_manager(info, NULL, context,
        G_SPAWN_SEARCH_PATH | G_SPAWN_DO_NOT_REAP_CHILD,
        NULL, NULL, pika_child_started, NULL, &error);
    char *message = ok ? NULL : g_strdup(error ? error->message : "Desktop launch failed");
    g_clear_error(&error);
    g_object_unref(context);
    g_object_unref(info);
    return message;
}
*/
import "C"

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"unsafe"
)

//export pikaDesktopChildStarted
func pikaDesktopChildStarted(pid C.int) {
	// G_SPAWN_DO_NOT_REAP_CHILD requires the parent to reap the child. Waiting
	// for this PID in Go also works in tests without a GLib main loop.
	go func() {
		child, err := os.FindProcess(int(pid))
		if err != nil {
			return
		}
		state, err := child.Wait()
		if err != nil {
			slog.Warn("desktop child wait failed", "error", err)
		} else if !state.Success() {
			slog.Info("desktop child exited", "status", state.ExitCode())
		}
	}()
}
func Launch(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("Desktop entry path must be absolute")
	}
	value := C.CString(path)
	defer C.free(unsafe.Pointer(value))
	message := C.pika_launch_desktop(value)
	if message != nil {
		defer C.g_free(C.gpointer(message))
		return fmt.Errorf("%s", C.GoString(message))
	}
	return nil
}
