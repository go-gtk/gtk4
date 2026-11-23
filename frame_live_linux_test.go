// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import (
	"bytes"
	"os"
	"testing"
)

func fill(pix []byte, x0, y0, x1, y1, w int, c [4]byte) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			copy(pix[(y*w+x)*4:], c[:])
		}
	}
}

// TestLiveSharedFrame proves a SharedFrame shows what was painted into Pix with
// no copy: frame A through SetFrame, then one rectangle repainted in place and
// presented through UpdateFrame, reads back as A with that rectangle changed; a
// second frame of another size replaces it whole (a resize); and nothing is
// copied, which the test checks by painting AFTER the texture was built and
// seeing the texture follow.
func TestLiveSharedFrame(t *testing.T) {
	ok, err := Init()
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if !ok {
		t.Skip("no display; run under Xvfb")
	}
	if NewSharedFrame(0, 4) != nil || NewSharedFrame(4, -1) != nil || NewSharedFrame(1<<20, 1<<20) != nil {
		t.Fatal("NewSharedFrame accepted a bad size")
	}
	const w, h = 16, 16
	f := NewSharedFrame(w, h)
	if f == nil || len(f.Pix()) != w*h*4 {
		t.Fatal("NewSharedFrame failed")
	}
	defer f.Release()
	fill(f.Pix(), 0, 0, w, h, w, [4]byte{10, 20, 30, 255})
	pic := PictureNew()
	rects := []Rect{{X: 6, Y: 4, W: 6, H: 4}}
	if pic.UpdateFrame(f, rects) {
		t.Fatal("UpdateFrame succeeded with no texture to update")
	}
	pic.SetFrame(f)
	a := append([]byte(nil), f.Pix()...)
	got := make([]byte, w*h*4)
	if !pic.ReadRGBA(got, w, h) || !bytes.Equal(got, a) {
		t.Fatal("SetFrame does not show the frame")
	}
	fill(f.Pix(), 6, 4, 12, 8, w, [4]byte{200, 100, 50, 255})
	b := append([]byte(nil), f.Pix()...)
	if !TextureUpdateSupported() {
		if os.Getenv("GTK4_REQUIRE_UPDATE") != "" {
			t.Fatalf("GTK4_REQUIRE_UPDATE is set but libgtk-4 4.%d lacks the update API", MinorVersion())
		}
		if pic.UpdateFrame(f, rects) {
			t.Fatal("UpdateFrame succeeded below GTK 4.16")
		}
		return
	}
	if !pic.UpdateFrame(f, rects) {
		t.Fatal("UpdateFrame refused a valid update")
	}
	if !pic.ReadRGBA(got, w, h) || !bytes.Equal(got, b) {
		t.Fatal("picture after UpdateFrame differs from the frame")
	}
	if pic.UpdateFrame(f, []Rect{{X: 100, Y: 100, W: 4, H: 4}}) {
		t.Fatal("UpdateFrame accepted an off-frame rectangle")
	}

	g := NewSharedFrame(w+8, h+4)
	defer g.Release()
	fill(g.Pix(), 0, 0, w+8, h+4, w+8, [4]byte{1, 2, 3, 255})
	if pic.UpdateFrame(g, []Rect{{X: 0, Y: 0, W: 2, H: 2}}) {
		t.Fatal("UpdateFrame accepted a frame of another size than the texture")
	}
	pic.SetFrame(g)
	big := make([]byte, (w+8)*(h+4)*4)
	if !pic.ReadRGBA(big, w+8, h+4) || !bytes.Equal(big, g.Pix()) {
		t.Fatal("SetFrame of a resized frame does not show it")
	}
	// The first frame's memory is still referenced by nothing but this test: a
	// texture released by the picture must not have freed it from under Pix.
	f.Pix()[0] = 1
}

// TestLiveOverlayAllocatedSize proves a window around an overlay whose child is a
// picture lays out and reports a nonzero size through AllocatedSize.
func TestLiveOverlayAllocatedSize(t *testing.T) {
	ok, err := Init()
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if !ok {
		t.Skip("no display; run under Xvfb")
	}
	win := WindowNew()
	win.SetDefaultSize(200, 120)
	ov := OverlayNew()
	pic := PictureNew()
	ov.SetOverlayChild(pic.Widget())
	ov.AddOverlay(FixedNew())
	win.SetChild(ov)
	if w, h := pic.Widget().AllocatedSize(); w != 0 || h != 0 {
		t.Fatalf("allocated size before layout = %dx%d, want 0x0", w, h)
	}
	win.Present()
	loop := MainLoopNew()
	n := 0
	win.AddTickCallback(func() bool {
		n++
		if n == 30 {
			loop.Quit()
		}
		return true
	})
	loop.Run()
	if w, h := pic.Widget().AllocatedSize(); w != 200 || h != 120 {
		t.Fatalf("picture allocated %dx%d, want it to fill the 200x120 window", w, h)
	}
}
