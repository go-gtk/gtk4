// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import (
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ebitengine/purego"
)

var (
	tdOnce sync.Once

	tdToplevels  func() uintptr
	tdFirstChild func(uintptr) uintptr
	tdNextSib    func(uintptr) uintptr
	tdButtonType func() uint64
	tdButtonText func(uintptr) uintptr
	tdClose      func(uintptr)
	tdFileNew    func(string) uintptr
	tdSetFile    func(uintptr, uintptr, uintptr) bool
	tdSetFolder  func(uintptr, uintptr, uintptr) bool
	tdTimeout    func(uint32, uintptr, uintptr) uint32
	tdClicked    func(uintptr)
)

func tdLoad(t *testing.T) {
	t.Helper()
	tdOnce.Do(func() {
		if err := loadMenu(); err != nil { // gtkWidgetGetMapped
			t.Fatal(err)
		}
		gtk, _ := dlopenAny("libgtk-4.so.1")
		glib, _ := dlopenAny("libglib-2.0.so.0")
		gio, _ := dlopenAny("libgio-2.0.so.0")
		reg := func(p any, h uintptr, n string) { purego.RegisterLibFunc(p, h, n) }
		reg(&tdToplevels, gtk, "gtk_window_get_toplevels")
		reg(&tdFirstChild, gtk, "gtk_widget_get_first_child")
		reg(&tdNextSib, gtk, "gtk_widget_get_next_sibling")
		reg(&tdButtonType, gtk, "gtk_button_get_type")
		reg(&tdButtonText, gtk, "gtk_button_get_label")
		reg(&tdClose, gtk, "gtk_window_close")
		reg(&tdFileNew, gio, "g_file_new_for_path")
		reg(&tdSetFile, gtk, "gtk_file_chooser_set_file")
		reg(&tdSetFolder, gtk, "gtk_file_chooser_set_current_folder")
		reg(&tdTimeout, glib, "g_timeout_add")
		reg(&tdClicked, gtk, "gtk_widget_activate")
	})
}

// toplevelsOtherThan lists the open windows that are not win.
func toplevelsOtherThan(win Widget) []uintptr {
	m := tdToplevels()
	var out []uintptr
	for i, n := uint32(0), lmCount(m); i < n; i++ {
		w := lmItem(m, i)
		if w != uintptr(win) && gtkWidgetGetMapped(w) {
			out = append(out, w)
		}
		dlgObjectUnref(w)
	}
	return out
}

// buttons returns every GtkButton under w, in tree order, with its label.
func buttons(w uintptr) (btns []uintptr, labels []string) {
	var walk func(uintptr)
	walk = func(n uintptr) {
		for c := tdFirstChild(n); c != 0; c = tdNextSib(c) {
			if gTypeCheckInstanceIsA(c, tdButtonType()) {
				btns = append(btns, c)
				labels = append(labels, cString(tdButtonText(c)))
			}
			walk(c)
		}
	}
	walk(w)
	return
}

// drive runs the main loop until the dialog callback calls finish, and once a
// dialog other than win has been on screen for a few ticks calls act on it. It
// fails the test on a 20 s timeout.
func drive(t *testing.T, win Widget, start func(finish func()), act func(dlg uintptr)) {
	t.Helper()
	tdLoad(t)
	// A window of an earlier dialog may still be on its way out: only one that
	// was not there before start is this dialog's.
	stale := map[uintptr]bool{}
	for _, d := range toplevelsOtherThan(win) {
		stale[d] = true
	}
	loop := MainLoopNew()
	finished := false
	start(func() { finished = true; loop.Quit() })

	deadline := time.Now().Add(20 * time.Second)
	seen, acted := 0, false
	cb := purego.NewCallback(func(_ uintptr) int32 {
		if finished {
			return 0
		}
		if time.Now().After(deadline) {
			t.Error("timeout: the dialog never completed")
			loop.Quit()
			return 0
		}
		if acted {
			return 1
		}
		var ds []uintptr
		for _, d := range toplevelsOtherThan(win) {
			if !stale[d] {
				ds = append(ds, d)
			}
		}
		if len(ds) > 0 {
			if seen++; seen >= 6 { // mapped for ~300 ms: let it populate
				acted = true
				act(ds[0])
			}
		}
		return 1
	})
	retainCallback(cb)
	tdTimeout(50, cb, 0)
	loop.Run()
	pump(30) // let the finished dialog's window go
}

func liveParent(t *testing.T) Widget {
	t.Helper()
	win, _ := liveWindow(t)
	win.Present()
	pump(20)
	if !DialogsSupported() {
		t.Skip("GTK older than 4.10: no GtkFileDialog/GtkAlertDialog")
	}
	return win
}

func TestLiveAlertButtons(t *testing.T) {
	win := liveParent(t)
	for want := 0; want < 3; want++ {
		var got AlertResult
		calls := 0
		drive(t, win, func(finish func()) {
			err := ShowAlert(win, AlertRequest{Message: "Delete?", Detail: "d", Buttons: []string{"Yes", "No", "Maybe"}},
				func(r AlertResult) { got = r; calls++; finish() })
			if err != nil {
				t.Fatalf("ShowAlert: %v", err)
			}
		}, func(dlg uintptr) {
			btns, labels := buttons(dlg)
			if !reflect.DeepEqual(labels, []string{"Yes", "No", "Maybe"}) {
				t.Errorf("button labels %q", labels)
				tdClose(dlg)
				return
			}
			tdClicked(btns[want])
		})
		if got.Err != nil || got.Cancelled || got.Button != want || calls != 1 {
			t.Fatalf("pressed %d: %+v after %d calls", want, got, calls)
		}
	}
}

func TestLiveAlertDismissed(t *testing.T) {
	win := liveParent(t)
	var got AlertResult
	drive(t, win, func(finish func()) {
		if err := ShowAlert(win, AlertRequest{Message: "m", Buttons: []string{"A", "B"}},
			func(r AlertResult) { got = r; finish() }); err != nil {
			t.Fatal(err)
		}
	}, func(dlg uintptr) { tdClose(dlg) })
	if !got.Cancelled || got.Err != nil {
		t.Fatalf("closed alert: %+v, want Cancelled", got)
	}
}

func TestLiveFileDialogCancel(t *testing.T) {
	win := liveParent(t)
	for _, k := range []FileDialogKind{FileOpen, FileOpenMultiple, FileSelectFolder, FileSave} {
		var got FileDialogResult
		drive(t, win, func(finish func()) {
			if err := ShowFileDialog(win, FileDialogRequest{Kind: k, Title: "t", InitialName: "x.txt",
				Filters: []FileFilter{{Name: "Text", Suffixes: []string{"txt"}}}},
				func(r FileDialogResult) { got = r; finish() }); err != nil {
				t.Fatal(err)
			}
		}, func(dlg uintptr) { tdClose(dlg) })
		if !got.Cancelled || got.Err != nil || len(got.Paths) != 0 {
			t.Fatalf("kind %d closed: %+v, want Cancelled", k, got)
		}
	}
}

func TestLiveFileDialogChoose(t *testing.T) {
	win := liveParent(t)
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(req FileDialogRequest, act func(dlg uintptr)) FileDialogResult {
		var got FileDialogResult
		drive(t, win, func(finish func()) {
			if err := ShowFileDialog(win, req, func(r FileDialogResult) { got = r; finish() }); err != nil {
				t.Fatal(err)
			}
		}, act)
		return got
	}
	file := func(p string) uintptr { return tdFileNew(p) }
	accept := func(dlg uintptr) { // press the dialog's default action button, as a person would
		btns, labels := buttons(dlg)
		for i, l := range labels {
			if l == "_Open" || l == "_Select" || l == "_Save" {
				tdClicked(btns[i])
				return
			}
		}
		t.Errorf("no accept button among %q", labels)
		tdClose(dlg)
	}

	got := run(FileDialogRequest{Kind: FileOpen}, func(dlg uintptr) {
		tdSetFile(dlg, file(a), 0)
		accept(dlg)
	})
	if got.Err != nil || !reflect.DeepEqual(got.Paths, []string{a}) {
		t.Fatalf("open: %+v", got)
	}

	got = run(FileDialogRequest{Kind: FileOpenMultiple}, func(dlg uintptr) {
		// GTK 4 has no gtk_file_chooser_select_file: one selection is what a
		// test can set, and the list API must return it as a one-element list.
		tdSetFile(dlg, file(b), 0)
		accept(dlg)
	})
	if got.Err != nil || !reflect.DeepEqual(got.Paths, []string{b}) {
		t.Fatalf("open multiple: %+v", got)
	}

	got = run(FileDialogRequest{Kind: FileSelectFolder}, func(dlg uintptr) {
		tdSetFile(dlg, file(dir), 0)
		accept(dlg)
	})
	if got.Err != nil || !reflect.DeepEqual(got.Paths, []string{dir}) {
		t.Fatalf("folder: %+v", got)
	}

	got = run(FileDialogRequest{Kind: FileSave, InitialName: "new.txt"}, func(dlg uintptr) {
		tdSetFolder(dlg, file(dir), 0)
		accept(dlg)
	})
	if want := filepath.Join(dir, "new.txt"); got.Err != nil || !reflect.DeepEqual(got.Paths, []string{want}) {
		t.Fatalf("save: %+v, want %q", got, want)
	}
}
