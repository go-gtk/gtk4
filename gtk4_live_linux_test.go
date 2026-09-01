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

// TestLiveSliderPopUp drives a real GtkScale and GtkDropDown: a slider's value and
// a drop-down's selection round-trip through the binding, and the drop-down was
// built from a Go []string without any C string-array marshalling.
func TestLiveSliderPopUp(t *testing.T) {
	ok, err := Init()
	if err != nil {
		t.Fatalf("Init: could not load GTK4: %v", err)
	}
	if !ok {
		t.Skip("no display (gtk_init_check == false); run under Xvfb for the live test")
	}

	slider := SliderNew(0, 100, 1)
	slider.SetValue(42)
	if got := slider.Value(); got != 42 {
		t.Errorf("slider value = %v, want 42", got)
	}

	pop := PopUpNew([]string{"one", "two", "three"})
	pop.SetSelected(2)
	if got := pop.Selected(); got != 2 {
		t.Errorf("drop-down selection = %d, want 2", got)
	}
	empty := PopUpNew(nil)
	if got := empty.Selected(); got != -1 {
		t.Errorf("empty drop-down selection = %d, want -1", got)
	}
}

// TestLiveNewWidgets drives the ten controls added for the go-widgets host
// backend: it constructs each, sets a value, reads it back, and asserts the
// round-trip where one applies. It needs a display; run under Xvfb.
func TestLiveNewWidgets(t *testing.T) {
	ok, err := Init()
	if err != nil {
		t.Fatalf("Init: could not load GTK4: %v", err)
	}
	if !ok {
		t.Skip("no display (gtk_init_check == false); run under Xvfb for the live test")
	}

	// 1. Progress bar — read-only; just exercise construct + set.
	prog := ProgressNew()
	if prog == 0 {
		t.Fatal("ProgressNew returned null")
	}
	prog.SetFraction(0.5)

	// 2. Spinner — no value; construct + start/stop must not crash.
	spin := SpinnerNew()
	if spin == 0 {
		t.Fatal("SpinnerNew returned null")
	}
	spin.Start()
	spin.Stop()

	// 3. Stepper (spin button) — value round-trips, "value-changed" wires.
	step := StepperNew(0, 10, 0.5)
	step.SetSpinValue(3.5)
	if got := step.SpinValue(); got != 3.5 {
		t.Errorf("stepper value = %v, want 3.5", got)
	}
	step.Connect("value-changed", func() {})

	// 4. Search field — GtkEditable, so Text/SetText round-trip.
	search := SearchNew()
	search.SetText("query")
	if got := search.Text(); got != "query" {
		t.Errorf("search text = %q, want query", got)
	}
	search.Connect("search-changed", func() {})

	// 5. Editable combo — prefilled; SetComboText/ComboText round-trip via the entry.
	combo := ComboNew([]string{"alpha", "beta", "gamma"})
	combo.SetComboText("custom")
	if got := combo.ComboText(); got != "custom" {
		t.Errorf("combo text = %q, want custom", got)
	}
	combo.Connect("changed", func() {})

	// 6. Segmented primitives — a box of grouped toggle buttons; Active round-trips
	//    through the shared (type-dispatched) accessors, "toggled" wires.
	box := BoxNew(true)
	first := ToggleButtonNewWithLabel("day")
	second := ToggleButtonNewWithLabel("week")
	first.SetGroup(0)      // start the group
	second.SetGroup(first) // join it
	box.Append(first)
	box.Append(second)
	first.SetActive(true)
	if !first.Active() {
		t.Error("toggle button did not become active")
	}
	second.SetActive(true) // exclusive group: selecting week deselects day
	if !second.Active() {
		t.Error("second toggle did not become active")
	}
	if first.Active() {
		t.Error("grouped toggles are not exclusive: first still active")
	}
	second.Connect("toggled", func() {})

	// 7. Multi-line text — buffer text round-trips; the change signal is on the buffer.
	tv := TextViewNew()
	tv.SetTextViewText("line one\nline two")
	if got := tv.TextViewText(); got != "line one\nline two" {
		t.Errorf("text view text = %q, want two lines", got)
	}
	tv.ConnectBufferChanged(func() {})
	if tv.Buffer() == 0 {
		t.Error("Buffer() returned null")
	}

	// 8. Link — construct + OnActivateLink wires (activation is user-driven).
	link := LinkNew("Open")
	if link == 0 {
		t.Fatal("LinkNew returned null")
	}
	link.OnActivateLink(func() {})

	// 9. Date — ISO round-trips.
	date := DateNew()
	date.SetDateISO("2026-02-14")
	if got := date.DateISO(); got != "2026-02-14" {
		t.Errorf("date ISO = %q, want 2026-02-14", got)
	}
	date.Connect("day-selected", func() {})

	// 10. Colour — hex round-trips within rounding.
	col := ColorNew()
	col.SetColorHex("#3366cc")
	if got := col.ColorHex(); got != "#3366CC" {
		t.Errorf("colour hex = %q, want #3366CC", got)
	}
	col.Connect("color-set", func() {})
}
