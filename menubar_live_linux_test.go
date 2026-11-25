// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import (
	"os"
	"testing"
)

// TestLiveMenuBar proves, on a real GTK4, that a bar of nested menus is built on
// a plain GtkWindow, that choosing a row by its action name reaches the Go
// callback, that a check row keeps the host's state, and that a disabled row is
// refused. With GTK4_REQUIRE_DISPLAY set a missing display fails instead of
// skipping, so a CI lane cannot go green without running it.
func TestLiveMenuBar(t *testing.T) {
	ok, err := Init()
	if err != nil || !ok {
		if os.Getenv("GTK4_REQUIRE_DISPLAY") != "" {
			t.Fatalf("no display (ok=%v err=%v) and GTK4_REQUIRE_DISPLAY is set", ok, err)
		}
		t.Skip("no display; run under Xvfb")
	}
	win := WindowNew()
	win.SetDefaultSize(400, 240)
	box := BoxNew(false)
	win.SetChild(box)
	content := FixedNew()
	content.SetExpand(true, true)
	box.Append(content)

	bar := MenuBarNew(win)
	if bar == nil {
		t.Fatal("MenuBarNew returned nil")
	}
	box.Prepend(bar.Widget())

	var got []string
	checked := false
	bar.SetItems([]BarItem{
		{Label: "File", Sub: []BarItem{
			{Label: "Open", Accel: "<Control>o", OnActivate: func() { got = append(got, "Open") }},
			{Separator: true},
			{Label: "Recent", Sub: []BarItem{{Label: "a", OnActivate: func() { got = append(got, "a") }}}},
			{Label: "Quit", OnActivate: func() { got = append(got, "Quit") }},
		}},
		{Label: "View", Sub: []BarItem{
			{Label: "Ruler", Check: true, Checked: checked, OnActivate: func() { got = append(got, "Ruler") }},
			{Label: "Inert"},
		}},
	})
	if bar.Len() != 5 {
		t.Fatalf("Len = %d, want 5", bar.Len())
	}
	win.Present()
	pump(20)

	for _, i := range []int{0, 1, 2} {
		if !bar.Activate(i) {
			t.Errorf("Activate(%d) refused a live row", i)
		}
	}
	if bar.Activate(4) {
		t.Error("Activate accepted the inert row")
	}
	if !bar.Activate(3) {
		t.Error("Activate refused the check row")
	}
	want := []string{"Open", "a", "Quit", "Ruler"}
	if len(got) != len(want) {
		t.Fatalf("callbacks = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("callbacks = %v, want %v", got, want)
		}
	}

	// A swap leaves a working bar.
	got = nil
	bar.SetItems([]BarItem{{Label: "File", Sub: []BarItem{{Label: "New", OnActivate: func() { got = append(got, "New") }}}}})
	if !bar.Activate(0) || len(got) != 1 {
		t.Errorf("after SetItems: callbacks = %v", got)
	}
	bar.Destroy()
	win.SetVisible(false)
}
