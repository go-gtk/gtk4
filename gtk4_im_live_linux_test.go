// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import (
	"testing"
	"unsafe"
)

// TestCStringStopsAtTheTerminator reads the C strings GTK hands the binding: a
// terminated string, the empty string, the null pointer, and a buffer with no
// terminator inside the bound (which must stop, not run on).
func TestCStringStopsAtTheTerminator(t *testing.T) {
	addr := func(b []byte) uintptr { return uintptr(unsafe.Pointer(&b[0])) }
	if got := cString(addr([]byte("ê\x00junk"))); got != "ê" {
		t.Errorf("cString = %q, want %q", got, "ê")
	}
	if got := cString(addr([]byte{0})); got != "" {
		t.Errorf("empty string = %q", got)
	}
	if got := cString(0); got != "" {
		t.Errorf("null pointer = %q", got)
	}
	long := make([]byte, maxCString+10)
	for i := range long {
		long[i] = 'x'
	}
	if got := len(cString(addr(long))); got != maxCString {
		t.Errorf("unterminated string read %d bytes, want the bound %d", got, maxCString)
	}
}

// TestLiveIMContext drives a real GtkIMContextSimple and a GtkIMMulticontext
// through the binding: both construct, take a client widget, focus, a cursor
// location and a reset, report an empty preedit when nothing is composed, and
// attach to a key controller. The compose sequence itself needs key events from
// the display server, so it is proven where a window and a key injector exist
// (the go-widgets/window lane), not here.
func TestLiveIMContext(t *testing.T) {
	ok, err := Init()
	if err != nil {
		t.Fatalf("Init: could not load GTK4: %v", err)
	}
	if !ok {
		t.Skip("no display (gtk_init_check == false); run under Xvfb for the live test")
	}
	win := WindowNew()
	win.SetDefaultSize(200, 120)
	win.SetChild(FixedNew())
	for name, im := range map[string]IMContext{"simple": IMContextSimpleNew(), "multi": IMContextNew()} {
		if im == 0 {
			t.Fatalf("%s: no context", name)
		}
		im.SetClientWidget(win)
		im.OnCommit(func(string) {})
		im.OnPreeditChanged(func() {})
		win.OnKeyIM(im, func(uint, uint, uint, bool) {})
		win.Present()
		im.FocusIn()
		im.SetCursorLocation(10, 20, 2, 16)
		if text, cur := im.Preedit(); text != "" || cur != 0 {
			t.Errorf("%s: preedit = %q, %d; want none", name, text, cur)
		}
		im.Reset()
		im.FocusOut()
	}
}
