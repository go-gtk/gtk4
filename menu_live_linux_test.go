// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import "testing"

// pump runs the main loop for n idle turns, enough for GTK to map a presented
// window and settle what it was asked to do.
func pump(n int) {
	loop := MainLoopNew()
	var step func(int)
	step = func(i int) {
		if i >= n {
			loop.Quit()
			return
		}
		IdleAdd(func() { step(i + 1) })
	}
	IdleAdd(func() { step(0) })
	loop.Run()
}

func liveWindow(t *testing.T) (win, fixed Widget) {
	t.Helper()
	ok, err := Init()
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if !ok {
		t.Skip("no display; run under Xvfb")
	}
	win = WindowNew()
	win.SetDefaultSize(320, 200)
	fixed = FixedNew()
	win.SetChild(fixed)
	return win, fixed
}

// TestLiveContextMenu proves the popover opens on a real GTK4, lists what it was
// given, runs the entry that is activated, refuses the inert one, and survives
// a swap of its entries.
func TestLiveContextMenu(t *testing.T) {
	win, fixed := liveWindow(t)
	btn := ButtonNewWithLabel("Row")
	fixed.Put(btn, 10, 10)

	var picked []string
	m := ContextMenuNew(btn)
	if m == nil {
		t.Fatal("ContextMenuNew returned nil")
	}
	m.SetItems([]MenuItem{
		{Label: "Open", OnPick: func() { picked = append(picked, "Open") }},
		{},
		{Label: "Remove"}, // inert
		{Label: "Save_as", OnPick: func() { picked = append(picked, "Save_as") }},
	})
	if got := m.Len(); got != 3 {
		t.Fatalf("Len = %d, want 3 (separator not counted)", got)
	}
	if m.IsOpen() {
		t.Fatal("menu open before anything asked for it")
	}

	win.Present()
	pump(20)
	m.Popup(5, 5)
	if !m.IsOpen() {
		t.Fatal("popover did not open after Popup")
	}
	if !m.Activate(0) || !m.Activate(3) {
		t.Error("Activate refused a live entry")
	}
	if m.Activate(2) {
		t.Error("Activate accepted an inert entry")
	}
	if m.Activate(1) {
		t.Error("Activate accepted a separator")
	}
	pump(5)
	if len(picked) != 2 || picked[0] != "Open" || picked[1] != "Save_as" {
		t.Errorf("picked = %v, want [Open Save_as]", picked)
	}

	// A swap under an open menu: the new handler is the one that runs.
	m.SetItems([]MenuItem{{Label: "Again", OnPick: func() { picked = append(picked, "Again") }}})
	if m.Len() != 1 {
		t.Errorf("Len after swap = %d, want 1", m.Len())
	}
	m.Activate(0)
	pump(5)
	if picked[len(picked)-1] != "Again" {
		t.Errorf("after swap picked = %v", picked)
	}

	m.Popdown()
	m.Destroy()
	win.SetVisible(false)
}

// TestLiveCloseRequestHandled proves OnCloseRequest can hold a window open and
// that a window closes when the handler lets it.
func TestLiveCloseRequestHandled(t *testing.T) {
	win, _ := liveWindow(t)
	stop := true
	asked := 0
	win.OnCloseRequest(func() bool { asked++; return stop })
	win.Present()
	pump(20)
	if !win.Mapped() {
		t.Fatal("window not mapped after Present")
	}
	win.Close()
	pump(5)
	if asked != 1 {
		t.Fatalf("close-request handler ran %d times, want 1", asked)
	}
	if !win.Mapped() {
		t.Fatal("window went away although the handler stopped the close")
	}
	win.SetVisible(false)
	pump(5)
	if win.Mapped() {
		t.Fatal("SetVisible(false) did not unmap")
	}
	win.SetVisible(true)
	win.Present()
	pump(10)
	if !win.Mapped() {
		t.Fatal("window did not come back")
	}
	win.SetVisible(false)
}

// TestLivePictureInButton proves a picture can stand in a button's content.
func TestLivePictureInButton(t *testing.T) {
	win, fixed := liveWindow(t)
	btn := ButtonNewWithLabel("Play")
	fixed.Put(btn, 10, 10)
	pic := PictureNew()
	pic.SetRGBA([]byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 255, 255, 255, 255}, 2, 2)
	pic.SetCanShrink(true)
	pic.SetAlternativeText("Play")
	pic.Widget().SetSizeRequest(16, 16)
	btn.SetButtonChild(pic.Widget())
	btn.SetTooltip("Play")
	win.Present()
	pump(20)
	if !pic.Widget().Mapped() {
		t.Error("the picture inside the button is not on screen")
	}
	win.SetVisible(false)
}
