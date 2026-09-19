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
static gint pika_shadow_margin;

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

// Shadows are visual only: clicks in the new transparent gutter pass through.
static void pika_content_input_region(GtkWidget *widget, GtkAllocation *allocation, gpointer unused) {
    GdkWindow *native = gtk_widget_get_window(widget);
    if (!native) return;
    cairo_rectangle_int_t content = {pika_shadow_margin, pika_shadow_margin,
        MAX(0, allocation->width - 2 * pika_shadow_margin),
        MAX(0, allocation->height - 2 * pika_shadow_margin)};
    cairo_region_t *region = cairo_region_create_rectangle(&content);
    gdk_window_input_shape_combine_region(native, region, 0, 0);
    cairo_region_destroy(region);
}

// Wails 2.15 queues StartHidden's hide but synchronously calls show_all.
// Guard that first map before WebKit has painted, including --background.
static gboolean pika_first_realize(GSignalInvocationHint *hint, guint count, const GValue *values, gpointer unused) {
    GtkWidget *widget = g_value_get_object(&values[0]);
    // Wails also queues the title assignment, so it is not available yet.
    // Match the process-local toplevel containing our WebKit view instead.
    if (!GTK_IS_WINDOW(widget) || !pika_webview(widget)) return TRUE;
    gtk_widget_set_opacity(widget, 0.0);
    gtk_window_set_focus_on_map(GTK_WINDOW(widget), FALSE);
    gtk_window_set_accept_focus(GTK_WINDOW(widget), FALSE);
    GtkCssProvider *css = gtk_css_provider_new();
    gtk_css_provider_load_from_data(css, "window { background-color: transparent; background-image: none; }", -1, NULL);
    gtk_style_context_add_provider(gtk_widget_get_style_context(widget), GTK_STYLE_PROVIDER(css), GTK_STYLE_PROVIDER_PRIORITY_APPLICATION);
    g_object_unref(css);
    g_signal_connect(widget, "size-allocate", G_CALLBACK(pika_content_input_region), NULL);
    GtkAllocation allocation;
    gtk_widget_get_allocation(widget, &allocation);
    pika_content_input_region(widget, &allocation, NULL);
    return FALSE; // Only the first realization of our own launcher.
}
static void pika_install_presentation(gint margin) {
    pika_shadow_margin = margin;
    gpointer klass = g_type_class_ref(GTK_TYPE_WIDGET);
    g_signal_add_emission_hook(g_signal_lookup("realize", GTK_TYPE_WIDGET), 0, pika_first_realize, NULL, NULL);
    g_type_class_unref(klass);
}
static gboolean pika_present_on_main(gpointer unused) {
    GtkWindow *window = pika_window();
    if (window) gtk_widget_set_opacity(GTK_WIDGET(window), 1.0);
    return G_SOURCE_REMOVE;
}
static void pika_present(void) { g_idle_add(pika_present_on_main, NULL); }

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
        if (gtk_widget_get_opacity(GTK_WIDGET(window)) > 0.99) flags |= 8;
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

func focusLauncher()               { C.pika_request_focus() }
func cancelLauncherFocus()         { C.pika_cancel_focus() }
func installLauncherPresentation() { C.pika_install_presentation(C.gint(shadowMargin)) }
func presentLauncher()             { C.pika_present() }
func nativeFocusState() (mapped, active, webview, surfaceReady bool) {
	focusSnapshotMu.Lock()
	defer focusSnapshotMu.Unlock()
	C.pika_request_snapshot()
	deadline := time.Now().Add(500 * time.Millisecond)
	for C.pika_snapshot_done() == 0 {
		if time.Now().After(deadline) {
			return false, false, false, false
		}
		time.Sleep(5 * time.Millisecond)
	}
	flags := int(C.pika_snapshot_flags())
	return flags&1 != 0, flags&2 != 0, flags&4 != 0, flags&8 != 0
}
