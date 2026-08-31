// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

// Package gtk4 binds the parts of GTK4 a windowing toolkit needs — windows, the
// common controls, GObject signals and the GLib main loop — from pure Go with
// CGO_ENABLED=0, over github.com/ebitengine/purego. It is the Linux sibling of
// github.com/go-macos/objc: the shared native-toolkit bridge a go-widgets host
// backend embeds real controls through.
//
// It links no cgo: libgtk-4, libgobject-2.0 and libglib-2.0 are dlopen'd by
// soname and every entry point is resolved with purego.RegisterLibFunc, exactly
// as the feasibility spike proved on a stock Linux GTK4. A GObject signal reaches
// Go through a purego.NewCallback, so a control's clicks and edits are ordinary
// Go closures.
package gtk4

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

// A Widget is a GTK widget — a GObject pointer. The zero value is the null
// widget. Methods on it are thin wrappers over the C API.
type Widget uintptr

// A MainLoop is a GLib main loop.
type MainLoop uintptr

var (
	loadOnce sync.Once
	loadErr  error

	gtkInitCheck            func() bool
	gtkWindowNew            func() uintptr
	gtkWindowSetTitle       func(uintptr, string)
	gtkWindowSetDefault     func(uintptr, int32, int32)
	gtkWindowSetChild       func(uintptr, uintptr)
	gtkWindowPresent        func(uintptr)
	gtkFixedNew             func() uintptr
	gtkFixedPut             func(uintptr, uintptr, float64, float64)
	gtkFixedMove            func(uintptr, uintptr, float64, float64)
	gtkButtonNewWithLabel   func(string) uintptr
	gtkButtonSetLabel       func(uintptr, string)
	gtkCheckButtonNewWith   func(string) uintptr
	gtkCheckButtonActive    func(uintptr) bool
	gtkCheckButtonSetActive func(uintptr, bool)
	gtkEntryNew             func() uintptr
	gtkEntrySetVisibility   func(uintptr, bool)
	gtkLabelNew             func(string) uintptr
	gtkEditableGetText      func(uintptr) string
	gtkEditableSetText      func(uintptr, string)
	gtkWidgetSetSizeReq     func(uintptr, int32, int32)
	gtkWidgetSetVisible     func(uintptr, bool)
	gtkWidgetUnparent       func(uintptr)

	gSignalConnectData func(uintptr, string, uintptr, uintptr, uintptr, int32) uint64
	gMainLoopNew       func(uintptr, bool) uintptr
	gMainLoopRun       func(uintptr)
	gMainLoopQuit      func(uintptr)
	gIdleAdd           func(uintptr, uintptr) uint32
	gtkWidgetAddTick   func(uintptr, uintptr, uintptr, uintptr) uint32

	gtkWidgetAddController func(uintptr, uintptr)
	gtkGestureClickNew     func() uintptr
	gtkGestureSingleSetBtn func(uintptr, uint32)
	gtkGestureSingleGetBtn func(uintptr) uint32
	gtkEventCtrlGetState   func(uintptr) uint32
	gtkEventCtrlMotionNew  func() uintptr
	gtkEventCtrlScrollNew  func(uint32) uintptr
	gtkEventCtrlKeyNew     func() uintptr
	gdkKeyvalToUnicode     func(uint32) uint32
)

func load() error {
	loadOnce.Do(func() {
		glib, err := dlopenAny("libglib-2.0.so.0", "libglib-2.0.so")
		if err != nil {
			loadErr = err
			return
		}
		gobj, err := dlopenAny("libgobject-2.0.so.0", "libgobject-2.0.so")
		if err != nil {
			loadErr = err
			return
		}
		gtk, err := dlopenAny("libgtk-4.so.1", "libgtk-4.so")
		if err != nil {
			loadErr = err
			return
		}
		reg := func(p any, h uintptr, name string) { purego.RegisterLibFunc(p, h, name) }
		reg(&gtkInitCheck, gtk, "gtk_init_check")
		reg(&gtkWindowNew, gtk, "gtk_window_new")
		reg(&gtkWindowSetTitle, gtk, "gtk_window_set_title")
		reg(&gtkWindowSetDefault, gtk, "gtk_window_set_default_size")
		reg(&gtkWindowSetChild, gtk, "gtk_window_set_child")
		reg(&gtkWindowPresent, gtk, "gtk_window_present")
		reg(&gtkFixedNew, gtk, "gtk_fixed_new")
		reg(&gtkFixedPut, gtk, "gtk_fixed_put")
		reg(&gtkFixedMove, gtk, "gtk_fixed_move")
		reg(&gtkButtonNewWithLabel, gtk, "gtk_button_new_with_label")
		reg(&gtkButtonSetLabel, gtk, "gtk_button_set_label")
		reg(&gtkCheckButtonNewWith, gtk, "gtk_check_button_new_with_label")
		reg(&gtkCheckButtonActive, gtk, "gtk_check_button_get_active")
		reg(&gtkCheckButtonSetActive, gtk, "gtk_check_button_set_active")
		reg(&gtkEntryNew, gtk, "gtk_entry_new")
		reg(&gtkEntrySetVisibility, gtk, "gtk_entry_set_visibility")
		reg(&gtkLabelNew, gtk, "gtk_label_new")
		reg(&gtkEditableGetText, gtk, "gtk_editable_get_text")
		reg(&gtkEditableSetText, gtk, "gtk_editable_set_text")
		reg(&gtkWidgetSetSizeReq, gtk, "gtk_widget_set_size_request")
		reg(&gtkWidgetSetVisible, gtk, "gtk_widget_set_visible")
		reg(&gtkWidgetUnparent, gtk, "gtk_widget_unparent")
		reg(&gSignalConnectData, gobj, "g_signal_connect_data")
		reg(&gMainLoopNew, glib, "g_main_loop_new")
		reg(&gMainLoopRun, glib, "g_main_loop_run")
		reg(&gMainLoopQuit, glib, "g_main_loop_quit")
		reg(&gIdleAdd, glib, "g_idle_add")
		reg(&gtkWidgetAddTick, gtk, "gtk_widget_add_tick_callback")
		reg(&gtkWidgetAddController, gtk, "gtk_widget_add_controller")
		reg(&gtkGestureClickNew, gtk, "gtk_gesture_click_new")
		reg(&gtkGestureSingleSetBtn, gtk, "gtk_gesture_single_set_button")
		reg(&gtkGestureSingleGetBtn, gtk, "gtk_gesture_single_get_current_button")
		reg(&gtkEventCtrlGetState, gtk, "gtk_event_controller_get_current_event_state")
		reg(&gtkEventCtrlMotionNew, gtk, "gtk_event_controller_motion_new")
		reg(&gtkEventCtrlScrollNew, gtk, "gtk_event_controller_scroll_new")
		reg(&gtkEventCtrlKeyNew, gtk, "gtk_event_controller_key_new")
		reg(&gdkKeyvalToUnicode, gtk, "gdk_keyval_to_unicode")
	})
	return loadErr
}

func dlopenAny(names ...string) (uintptr, error) {
	var last error
	for _, n := range names {
		if h, err := purego.Dlopen(n, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
			return h, nil
		} else {
			last = err
		}
	}
	return 0, fmt.Errorf("gtk4: dlopen %v: %w", names, last)
}

// Init loads GTK4 and initialises it, reporting whether a display could be
// opened. It is safe to call more than once. An error means the libraries could
// not be loaded at all (not merely that there is no display).
func Init() (ok bool, err error) {
	if err := load(); err != nil {
		return false, err
	}
	return gtkInitCheck(), nil
}

// WindowNew creates a top-level window.
func WindowNew() Widget                { return Widget(gtkWindowNew()) }
func (w Widget) SetTitle(title string) { gtkWindowSetTitle(uintptr(w), title) }
func (w Widget) SetDefaultSize(width, height int) {
	gtkWindowSetDefault(uintptr(w), int32(width), int32(height))
}
func (w Widget) SetChild(child Widget) { gtkWindowSetChild(uintptr(w), uintptr(child)) }
func (w Widget) Present()              { gtkWindowPresent(uintptr(w)) }

// FixedNew creates a GtkFixed — the container a host overlays native controls in
// at absolute positions, over the pixel drawing area.
func FixedNew() Widget                           { return Widget(gtkFixedNew()) }
func (f Widget) Put(child Widget, x, y float64)  { gtkFixedPut(uintptr(f), uintptr(child), x, y) }
func (f Widget) Move(child Widget, x, y float64) { gtkFixedMove(uintptr(f), uintptr(child), x, y) }

// ButtonNewWithLabel creates a push button.
func ButtonNewWithLabel(label string) Widget { return Widget(gtkButtonNewWithLabel(label)) }
func (w Widget) SetLabel(label string)       { gtkButtonSetLabel(uintptr(w), label) }

// CheckButtonNewWithLabel creates a labelled check button. GTK check buttons that
// share a group act as radios; a host groups them for a NativeRadio.
func CheckButtonNewWithLabel(label string) Widget { return Widget(gtkCheckButtonNewWith(label)) }
func (w Widget) Active() bool                     { return gtkCheckButtonActive(uintptr(w)) }
func (w Widget) SetActive(on bool)                { gtkCheckButtonSetActive(uintptr(w), on) }

// EntryNew creates a single-line text entry. Call SetVisibility(false) for a
// secure (password) entry.
func EntryNew() Widget                  { return Widget(gtkEntryNew()) }
func (w Widget) SetVisibility(vis bool) { gtkEntrySetVisibility(uintptr(w), vis) }

// LabelNew creates a static text label.
func LabelNew(text string) Widget { return Widget(gtkLabelNew(text)) }

// Text and SetText read and write an editable's text (entry, label via editable
// where applicable), through the GtkEditable interface.
func (w Widget) Text() string     { return gtkEditableGetText(uintptr(w)) }
func (w Widget) SetText(s string) { gtkEditableSetText(uintptr(w), s) }

// SetSizeRequest fixes a widget's size (a host sizes controls to the region it
// laid out).
func (w Widget) SetSizeRequest(width, height int) {
	gtkWidgetSetSizeReq(uintptr(w), int32(width), int32(height))
}

// SetVisible shows or hides a widget in place.
func (w Widget) SetVisible(vis bool) { gtkWidgetSetVisible(uintptr(w), vis) }

// Unparent removes a widget from its parent (a host reconciling controls away).
func (w Widget) Unparent() { gtkWidgetUnparent(uintptr(w)) }

// Connect wires a GObject signal (e.g. "clicked", "changed") to a Go func. The
// callback keeps the widget's value reachable through its own accessors, so the
// zero-argument closure is enough for the control signals a host cares about. The
// returned handler id is GObject's; a host rarely needs it.
//
// The callback is retained for the process life (like the go-macos target
// classes): GTK holds a C pointer to it, and letting Go collect it would leave a
// dangling call.
func (w Widget) Connect(signal string, fn func()) uint64 {
	cb := purego.NewCallback(func(_ uintptr, _ uintptr) { fn() })
	retainCallback(cb)
	return gSignalConnectData(uintptr(w), signal, cb, 0, 0, 0)
}

// MainLoopNew creates a GLib main loop.
func MainLoopNew() MainLoop { return MainLoop(gMainLoopNew(0, false)) }
func (l MainLoop) Run()     { gMainLoopRun(uintptr(l)) }
func (l MainLoop) Quit()    { gMainLoopQuit(uintptr(l)) }

// IdleAdd schedules fn to run once on the main loop and be removed. A host uses
// it to marshal work onto the GTK thread. fn runs on the main loop thread.
func IdleAdd(fn func()) {
	cb := purego.NewCallback(func(_ uintptr) int32 { fn(); return 0 })
	retainCallback(cb)
	gIdleAdd(cb, 0)
}

// AddTickCallback registers fn to run once per frame, driven by this widget's
// GdkFrameClock, for as long as fn returns true. It is the GTK-native animation
// tick: aligned to the display's refresh, and quiescent while the widget is
// unmapped (the frame clock does not run then), so an idle hidden window costs
// nothing. A host that renders its own pixels into a [Picture] uses it to present
// a fresh frame each vsync — unlike a one-shot [IdleAdd], which the frame clock
// would fire before the window is even mapped. fn runs on the main-loop thread;
// returning false removes the callback. The callback is retained for the process
// life, like [Widget.Connect]'s.
func (w Widget) AddTickCallback(fn func() bool) uint64 {
	cb := purego.NewCallback(func(_ uintptr, _ uintptr, _ uintptr) int32 {
		if fn() {
			return 1 // G_SOURCE_CONTINUE — keep ticking
		}
		return 0 // G_SOURCE_REMOVE
	})
	retainCallback(cb)
	return uint64(gtkWidgetAddTick(uintptr(w), cb, 0, 0))
}

// retainCallback keeps every callback alive for the process: GTK stores the C
// function pointer and may call it at any later time.
var (
	cbMu   sync.Mutex
	cbKeep []uintptr
)

func retainCallback(cb uintptr) {
	cbMu.Lock()
	cbKeep = append(cbKeep, cb)
	cbMu.Unlock()
}
