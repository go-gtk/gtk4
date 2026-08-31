// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import "github.com/ebitengine/purego"

// GDK modifier-state bits, as they arrive in a controller's current event state
// and in a key event's state argument. A host decodes the four the toolkit cares
// about; the lock and button-mask bits are left alone.
const (
	ModShift   uint = 1 << 0  // GDK_SHIFT_MASK
	ModControl uint = 1 << 2  // GDK_CONTROL_MASK
	ModAlt     uint = 1 << 3  // GDK_ALT_MASK (Mod1)
	ModSuper   uint = 1 << 26 // GDK_SUPER_MASK
)

// scrollBothAxes is GTK_EVENT_CONTROLLER_SCROLL_BOTH_AXES — report vertical and
// horizontal wheel/trackpad deltas.
const scrollBothAxes uint32 = 1<<0 | 1<<1

// KeyvalToUnicode returns the Unicode rune a GDK keyval produces, or 0 for a
// non-printable (named) key such as Return or an arrow. It is
// gdk_keyval_to_unicode, so a host maps named keys itself and takes the rune for
// the rest.
func KeyvalToUnicode(keyval uint) rune { return rune(gdkKeyvalToUnicode(uint32(keyval))) }

// The event controllers below are how a host that renders its own pixels into a
// [Picture] gets pointer and key input: GTK delivers nothing to the framebuffer
// on its own, so the host attaches these to the window and translates each
// callback into its toolkit's event. Coordinates are widget-local points
// (gdouble); the host scales them to its framebuffer. Every callback is retained
// for the process life, like [Widget.Connect]'s.

// OnMouseDown fires on a press of any mouse button over w, with the button
// (1 left, 2 middle, 3 right), the GDK modifier state, and the widget-local point.
func (w Widget) OnMouseDown(fn func(button int, state uint, x, y float64)) {
	w.clickGesture("pressed", fn)
}

// OnMouseUp fires on the matching release.
func (w Widget) OnMouseUp(fn func(button int, state uint, x, y float64)) {
	w.clickGesture("released", fn)
}

func (w Widget) clickGesture(signal string, fn func(button int, state uint, x, y float64)) {
	g := gtkGestureClickNew()
	gtkGestureSingleSetBtn(g, 0) // 0 = listen to every button, not just the primary
	gtkWidgetAddController(uintptr(w), g)
	cb := purego.NewCallback(func(_ uintptr, _ int32, x, y float64, _ uintptr) {
		fn(int(gtkGestureSingleGetBtn(g)), uint(gtkEventCtrlGetState(g)), x, y)
	})
	retainCallback(cb)
	gSignalConnectData(g, signal, cb, 0, 0, 0)
}

// OnMotion fires when the pointer moves over w, with the modifier state (a button
// mask distinguishes a drag) and the widget-local point.
func (w Widget) OnMotion(fn func(state uint, x, y float64)) {
	c := gtkEventCtrlMotionNew()
	gtkWidgetAddController(uintptr(w), c)
	cb := purego.NewCallback(func(_ uintptr, x, y float64, _ uintptr) {
		fn(uint(gtkEventCtrlGetState(c)), x, y)
	})
	retainCallback(cb)
	gSignalConnectData(c, "motion", cb, 0, 0, 0)
}

// OnScroll fires on a wheel or trackpad scroll over w, with the delta in both axes
// (units are wheel notches / trackpad steps) and the modifier state.
func (w Widget) OnScroll(fn func(dx, dy float64, state uint)) {
	c := gtkEventCtrlScrollNew(scrollBothAxes)
	gtkWidgetAddController(uintptr(w), c)
	cb := purego.NewCallback(func(_ uintptr, dx, dy float64, _ uintptr) int32 {
		fn(dx, dy, uint(gtkEventCtrlGetState(c)))
		return 1 // handled
	})
	retainCallback(cb)
	gSignalConnectData(c, "scroll", cb, 0, 0, 0)
}

// OnKey fires on a key press and release that reaches w — that is, one no focused
// child (a native entry) consumed first. It gives the GDK keyval and keycode, the
// modifier state, and whether this is a press. Use [KeyvalToUnicode] for the rune
// and a keyval table for named keys.
func (w Widget) OnKey(fn func(keyval, keycode, state uint, press bool)) {
	c := gtkEventCtrlKeyNew()
	gtkWidgetAddController(uintptr(w), c)
	pressed := purego.NewCallback(func(_ uintptr, keyval, keycode, state uint32, _ uintptr) int32 {
		fn(uint(keyval), uint(keycode), uint(state), true)
		return 1 // handled — the host owns keys its framebuffer area receives
	})
	retainCallback(pressed)
	gSignalConnectData(c, "key-pressed", pressed, 0, 0, 0)
	released := purego.NewCallback(func(_ uintptr, keyval, keycode, state uint32, _ uintptr) {
		fn(uint(keyval), uint(keycode), uint(state), false)
	})
	retainCallback(released)
	gSignalConnectData(c, "key-released", released, 0, 0, 0)
}
