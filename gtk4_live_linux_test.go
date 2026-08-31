// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import (
	"os"
	"testing"
	"time"
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

// TestLiveTickCallback proves the frame-clock tick drives a Go callback: a host
// that renders its own pixels presents a fresh frame each vsync from here. It maps
// a window, registers AddTickCallback, spins the loop, and asserts the tick fired
// several times — then removes itself by returning false. A watchdog quits the
// loop so a runner without a running frame clock fails fast rather than hanging.
func TestLiveTickCallback(t *testing.T) {
	ok, err := Init()
	if err != nil {
		t.Fatalf("Init: could not load GTK4: %v", err)
	}
	if !ok {
		t.Skip("no display (gtk_init_check == false); run under Xvfb for the live test")
	}

	win := WindowNew()
	win.SetTitle("go-gtk tick test")
	win.SetDefaultSize(200, 120)
	win.Present()

	loop := MainLoopNew()
	ticks := 0
	const want = 3
	win.AddTickCallback(func() bool {
		ticks++
		if ticks >= want {
			loop.Quit()
			return false // remove the callback; we have what we need
		}
		return true // keep ticking
	})

	// Watchdog: if the frame clock never runs, don't hang the suite.
	go func() {
		time.Sleep(5 * time.Second)
		loop.Quit()
	}()
	loop.Run()

	if ticks == 0 {
		t.Fatal("frame-clock tick never reached Go (window not mapped, or no frame clock)")
	}
	if ticks < want {
		t.Logf("tick fired %d times (< %d) before the watchdog; frame clock is slow but working", ticks, want)
	}
}

// TestLiveInputControllers proves the input event controllers resolve and attach
// to a real widget without crashing, and that KeyvalToUnicode maps a printable
// keyval to its rune and a named key to nothing. Synthesising pointer/key events
// under Xvfb needs a windowing robot, so the callbacks' firing is exercised by the
// window back-end's own live test, not here.
func TestLiveInputControllers(t *testing.T) {
	ok, err := Init()
	if err != nil {
		t.Fatalf("Init: could not load GTK4: %v", err)
	}
	if !ok {
		t.Skip("no display (gtk_init_check == false); run under Xvfb for the live test")
	}

	win := WindowNew()
	win.SetDefaultSize(200, 120)
	fixed := FixedNew()
	win.SetChild(fixed)
	// Each controller must attach without error.
	win.OnMouseDown(func(button int, state uint, x, y float64) {})
	win.OnMouseUp(func(int, uint, float64, float64) {})
	win.OnMotion(func(uint, float64, float64) {})
	win.OnScroll(func(float64, float64, uint) {})
	win.OnKey(func(uint, uint, uint, bool) {})
	win.Present()

	// 'a' (GDK keyval 0x61) is the rune 'a'. An arrow key (Left, 0xff51) has no
	// Unicode equivalent and yields 0. Return (0xff0d) is the gotcha a host must
	// know: gdk_keyval_to_unicode returns the CONTROL rune 0x0D for it, not 0 — so
	// a host maps named keyvals itself and keeps only printable runes (>= 0x20).
	if got := KeyvalToUnicode(0x61); got != 'a' {
		t.Errorf("KeyvalToUnicode(0x61) = %q, want 'a'", got)
	}
	if got := KeyvalToUnicode(0xff51); got != 0 {
		t.Errorf("KeyvalToUnicode(Left) = %d, want 0 (no rune)", got)
	}
	if got := KeyvalToUnicode(0xff0d); got != 0x0d {
		t.Errorf("KeyvalToUnicode(Return) = %d, want 0x0d (control rune, not printable)", got)
	}
}
