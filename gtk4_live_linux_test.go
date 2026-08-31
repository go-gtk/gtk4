// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android

package gtk4

import (
	"os"
	"testing"
)

// TestLiveGTK4 drives real GTK4 through the binding: it initialises GTK, builds a
// window with an entry, a button and a check button in a GtkFixed, wires signals,
// reads and writes values, and runs the GLib main loop, quitting itself from an
// idle callback. It needs a display; on a headless runner run it under Xvfb (see
// the Dockerfile). Without a display gtk_init_check reports false and the test
// skips rather than failing for the wrong reason.
func TestLiveGTK4(t *testing.T) {
	ok, err := Init()
	if err != nil {
		t.Fatalf("Init: could not load GTK4: %v", err)
	}
	if !ok {
		t.Skip("no display (gtk_init_check == false); run under Xvfb for the live test")
	}

	win := WindowNew()
	win.SetTitle("go-gtk live test")
	win.SetDefaultSize(320, 180)

	fixed := FixedNew()
	entry := EntryNew()
	secure := EntryNew()
	secure.SetVisibility(false) // password
	button := ButtonNewWithLabel("OK")
	check := CheckButtonNewWithLabel("Enable")

	fixed.Put(entry, 10, 10)
	fixed.Put(secure, 10, 44)
	fixed.Put(button, 10, 78)
	fixed.Put(check, 10, 112)
	win.SetChild(fixed)

	// Value round-trips.
	entry.SetText("hello")
	if got := entry.Text(); got != "hello" {
		t.Errorf("entry text = %q, want hello", got)
	}
	secure.SetText("s3cr3t")
	if got := secure.Text(); got != "s3cr3t" {
		t.Errorf("secure text = %q, want s3cr3t", got)
	}
	check.SetActive(true)
	if !check.Active() {
		t.Error("check button did not become active")
	}

	// Signals reach Go.
	clicked := 0
	button.Connect("clicked", func() { clicked++ })
	changed := 0
	entry.Connect("changed", func() { changed++ })

	win.Present()

	// Drive the loop briefly, then quit from an idle callback; also mutate the
	// entry so its "changed" signal fires into Go.
	loop := MainLoopNew()
	IdleAdd(func() {
		entry.SetText("world") // fires "changed"
		loop.Quit()
	})
	loop.Run()

	if changed == 0 {
		t.Error("entry 'changed' signal never reached Go")
	}
	if entry.Text() != "world" {
		t.Errorf("entry text after idle = %q, want world", entry.Text())
	}
	_ = clicked // click is user-driven; wiring is proven by 'changed' reaching Go
	_ = os.Getenv
}
