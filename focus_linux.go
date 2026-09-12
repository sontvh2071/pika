//go:build linux

package main

/*
#cgo pkg-config: gtk+-3.0
#include <gtk/gtk.h>
#ifdef GDK_WINDOWING_X11
#include <gdk/gdkx.h>
#endif

static gint pika_focus_generation;
static gint pika_focus_snapshot_done;
static gint pika_focus_snapshot_flags;

static GtkWidget *pika_webview(GtkWidget *widget) {
    GType type = g_type_from_name("WebKitWebView");
    if (type && G_TYPE_CHECK_INSTANCE_TYPE(widget, type)) return widget;
    if (!GTK_IS_CONTAINER(widget)) return NULL;
    GList *children = gtk_container_get_children(GTK_CONTAINER(widget));
    GtkWidget *found = NULL;
    for (GList *i = children; i && !found; i = i->next) found = pika_webview(GTK_WIDGET(i->data));
    g_list_free(children);
    return found;
}

// GTK's toplevel list contains only this process's windows.
static GtkWindow *pika_window(void) {
    GList *windows = gtk_window_list_toplevels();
    GtkWindow *found = NULL;
    for (GList *i = windows; i; i = i->next) {
        GtkWindow *window = GTK_WINDOW(i->data);
        if (g_strcmp0(gtk_window_get_title(window), "Pika") == 0 && pika_webview(GTK_WIDGET(window))) {
            found = window;
            break;
        }
    }
    g_list_free(windows);
    return found;
}

static gboolean pika_focus_on_main(gpointer data) {
    if (GPOINTER_TO_INT(data) != g_atomic_int_get(&pika_focus_generation)) return G_SOURCE_REMOVE;
    GtkWindow *window = pika_window();
    if (!window || !gtk_widget_get_visible(GTK_WIDGET(window))) return G_SOURCE_REMOVE;
    GtkWidget *webview = pika_webview(GTK_WIDGET(window));
    guint32 timestamp = GDK_CURRENT_TIME;
#ifdef GDK_WINDOWING_X11
    GdkWindow *native = gtk_widget_get_window(GTK_WIDGET(window));
    if (native && GDK_IS_X11_WINDOW(native)) {
        // IPC has no GTK key event. A stale/zero event time lets Cinnamon
        // reject activation even while the always-on-top window is visible.
        gdk_window_set_events(native, gdk_window_get_events(native) | GDK_PROPERTY_CHANGE_MASK);
        timestamp = gdk_x11_get_server_time(native);
        gdk_x11_window_set_user_time(native, timestamp);
    }
#endif
    gtk_window_set_accept_focus(window, TRUE);
    gtk_window_set_focus_on_map(window, TRUE);
    gtk_widget_grab_focus(webview);
    gtk_window_present_with_time(window, timestamp);
    return G_SOURCE_REMOVE;
}
static void pika_request_focus(void) {
    gint generation = g_atomic_int_add(&pika_focus_generation, 1) + 1;
    g_idle_add(pika_focus_on_main, GINT_TO_POINTER(generation));
}
static void pika_cancel_focus(void) { g_atomic_int_inc(&pika_focus_generation); }

static gboolean pika_read_focus_on_main(gpointer unused) {
    gint flags = 0;
    GtkWindow *window = pika_window();
    if (window) {
        if (gtk_widget_get_mapped(GTK_WIDGET(window))) flags |= 1;
        if (gtk_window_is_active(window)) flags |= 2;
        GtkWidget *webview = pika_webview(GTK_WIDGET(window));
        if (webview && gtk_window_get_focus(window) == webview) flags |= 4;
    }
    g_atomic_int_set(&pika_focus_snapshot_flags, flags);
    g_atomic_int_set(&pika_focus_snapshot_done, 1);
    return G_SOURCE_REMOVE;
}
static void pika_request_snapshot(void) {
    g_atomic_int_set(&pika_focus_snapshot_done, 0);
    g_idle_add(pika_read_focus_on_main, NULL);
}
static gint pika_snapshot_done(void) { return g_atomic_int_get(&pika_focus_snapshot_done); }
static gint pika_snapshot_flags(void) { return g_atomic_int_get(&pika_focus_snapshot_flags); }
*/
import "C"

import (
	"sync"
	"time"
)

var focusSnapshotMu sync.Mutex

func focusLauncher()       { C.pika_request_focus() }
func cancelLauncherFocus() { C.pika_cancel_focus() }
func nativeFocusState() (mapped, active, webview bool) {
	focusSnapshotMu.Lock()
	defer focusSnapshotMu.Unlock()
	C.pika_request_snapshot()
	deadline := time.Now().Add(500 * time.Millisecond)
	for C.pika_snapshot_done() == 0 {
		if time.Now().After(deadline) {
			return false, false, false
		}
		time.Sleep(5 * time.Millisecond)
	}
	flags := int(C.pika_snapshot_flags())
	return flags&1 != 0, flags&2 != 0, flags&4 != 0
}
