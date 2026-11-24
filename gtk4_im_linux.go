// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// An IMContext is a GtkIMContext: the object GTK routes key events through so
// that dead keys, the compose key and an input method (ibus, fcitx, ...) can turn
// several key presses into one character. A host that paints its own text fields
// needs one, because GTK only feeds it to its own entry widgets.
//
// The shape of the protocol, as GTK defines it: the host forwards key events
// (see [Widget.OnKeyIM], which does it), the context emits "preedit-changed"
// while a composition is under way — [IMContext.Preedit] then reads the
// uncommitted text — and "commit" with the finished text. The host tells the
// context where the caret is ([IMContext.SetCursorLocation]) so the input
// method can put its candidate window next to it.
type IMContext uintptr

var (
	imOnce sync.Once
	imErr  error

	gtkIMMultiNew     func() uintptr
	gtkIMSimpleNew    func() uintptr
	gtkIMSetClient    func(uintptr, uintptr)
	gtkIMFocusIn      func(uintptr)
	gtkIMFocusOut     func(uintptr)
	gtkIMReset        func(uintptr)
	gtkIMSetCursorLoc func(uintptr, uintptr)
	gtkIMGetPreedit   func(uintptr, uintptr, uintptr, uintptr)
	gtkEventCtrlKeyIM func(uintptr, uintptr)
	gIMUnref          func(uintptr)
	gFree             func(uintptr)
)

func loadIM() error {
	imOnce.Do(func() {
		if imErr = load(); imErr != nil {
			return
		}
		glib, err := dlopenAny("libglib-2.0.so.0", "libglib-2.0.so")
		if err != nil {
			imErr = err
			return
		}
		gobj, err := dlopenAny("libgobject-2.0.so.0", "libgobject-2.0.so")
		if err != nil {
			imErr = err
			return
		}
		gtk, err := dlopenAny("libgtk-4.so.1", "libgtk-4.so")
		if err != nil {
			imErr = err
			return
		}
		reg := func(p any, h uintptr, name string) { purego.RegisterLibFunc(p, h, name) }
		reg(&gtkIMMultiNew, gtk, "gtk_im_multicontext_new")
		reg(&gtkIMSimpleNew, gtk, "gtk_im_context_simple_new")
		reg(&gtkIMSetClient, gtk, "gtk_im_context_set_client_widget")
		reg(&gtkIMFocusIn, gtk, "gtk_im_context_focus_in")
		reg(&gtkIMFocusOut, gtk, "gtk_im_context_focus_out")
		reg(&gtkIMReset, gtk, "gtk_im_context_reset")
		reg(&gtkIMSetCursorLoc, gtk, "gtk_im_context_set_cursor_location")
		reg(&gtkIMGetPreedit, gtk, "gtk_im_context_get_preedit_string")
		reg(&gtkEventCtrlKeyIM, gtk, "gtk_event_controller_key_set_im_context")
		reg(&gIMUnref, gobj, "g_object_unref")
		reg(&gFree, glib, "g_free")
	})
	return imErr
}

// IMContextNew returns a GtkIMMulticontext, the context GTK's own entries use: it
// picks the input method the user's session selects (GTK_IM_MODULE, the
// desktop's setting) and falls back to the built-in simple one, which handles
// dead keys and compose sequences with no daemon. It returns 0 when GTK cannot
// be loaded.
func IMContextNew() IMContext {
	if loadIM() != nil {
		return 0
	}
	return IMContext(gtkIMMultiNew())
}

// IMContextSimpleNew returns a GtkIMContextSimple: dead keys and compose
// sequences only, never an input-method daemon. A host that wants the same
// behaviour on every machine, and a test that must not depend on whether ibus
// runs, use it instead of [IMContextNew].
func IMContextSimpleNew() IMContext {
	if loadIM() != nil {
		return 0
	}
	return IMContext(gtkIMSimpleNew())
}

// SetClientWidget names the widget the context composes for; GTK uses it to find
// the display and the surface an input method pops its candidate window on.
func (c IMContext) SetClientWidget(w Widget) {
	if c != 0 {
		gtkIMSetClient(uintptr(c), uintptr(w))
	}
}

// FocusIn tells the context its client has the keyboard focus. A context that is
// not focused ignores key events.
func (c IMContext) FocusIn() {
	if c != 0 {
		gtkIMFocusIn(uintptr(c))
	}
}

// FocusOut tells the context its client lost the keyboard focus.
func (c IMContext) FocusOut() {
	if c != 0 {
		gtkIMFocusOut(uintptr(c))
	}
}

// Reset abandons the composition in progress; the context emits
// "preedit-changed" once more, with an empty preedit.
func (c IMContext) Reset() {
	if c != 0 {
		gtkIMReset(uintptr(c))
	}
}

// SetCursorLocation tells the input method where the caret is, as a rectangle in
// the client widget's coordinates (points). It anchors the candidate window.
func (c IMContext) SetCursorLocation(x, y, w, h int) {
	if c == 0 {
		return
	}
	r := [4]int32{int32(x), int32(y), int32(w), int32(h)} // GdkRectangle
	gtkIMSetCursorLoc(uintptr(c), uintptr(unsafe.Pointer(&r[0])))
}

// Preedit returns the text being composed and the caret's position in it, in
// characters (not bytes). Text is empty when nothing is being composed. The
// Pango attributes — GTK's underline styling of the preedit — are not read: a
// host draws its own.
func (c IMContext) Preedit() (text string, cursor int) {
	if c == 0 {
		return "", 0
	}
	var s uintptr
	var cur int32
	gtkIMGetPreedit(uintptr(c), uintptr(unsafe.Pointer(&s)), 0, uintptr(unsafe.Pointer(&cur)))
	text = takeCString(s)
	return text, int(cur)
}

// OnPreeditChanged fires each time the composition changes: a character is added,
// a candidate is chosen, or the preedit ends. Read it with [IMContext.Preedit].
func (c IMContext) OnPreeditChanged(fn func()) {
	if c == 0 {
		return
	}
	cb := purego.NewCallback(func(_ uintptr, _ uintptr) { fn() })
	retainCallback(cb)
	gSignalConnectData(uintptr(c), "preedit-changed", cb, 0, 0, 0)
}

// OnCommit fires with the finished text of a composition (or an ordinary
// character the context passes through).
func (c IMContext) OnCommit(fn func(text string)) {
	if c == 0 {
		return
	}
	cb := purego.NewCallback(func(_ uintptr, s uintptr, _ uintptr) { fn(cString(s)) })
	retainCallback(cb)
	gSignalConnectData(uintptr(c), "commit", cb, 0, 0, 0)
}

// Free drops the host's reference.
func (c IMContext) Free() {
	if c != 0 {
		gIMUnref(uintptr(c))
	}
}

// OnKeyIM is [Widget.OnKey] with an input method in front: GTK offers each key
// event to im first, and fn sees only the events im did not consume — so a dead
// key or a compose sequence reaches im, and the character it makes reaches the
// host through [IMContext.OnCommit], while an ordinary letter the context passes
// through also arrives that way and is NOT delivered to fn. Named keys, shortcuts
// and releases still reach fn.
func (w Widget) OnKeyIM(im IMContext, fn func(keyval, keycode, state uint, press bool)) {
	w.keyController(im, fn)
}

// maxCString bounds the bytes read from a C string handed over by GTK: an input
// method's commit is a few characters, and a bug in it must not be an unbounded
// read.
const maxCString = 1 << 16

// cString copies the NUL-terminated string at address p, without freeing it.
func cString(p uintptr) string {
	if p == 0 {
		return ""
	}
	ptr := *(*unsafe.Pointer)(unsafe.Pointer(&p))
	n := 0
	for n < maxCString && *(*byte)(unsafe.Add(ptr, n)) != 0 {
		n++
	}
	return string(unsafe.Slice((*byte)(ptr), n))
}

// takeCString copies the string at p and releases it with g_free.
func takeCString(p uintptr) string {
	s := cString(p)
	if p != 0 {
		gFree(p)
	}
	return s
}
