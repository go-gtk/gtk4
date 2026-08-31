// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import "testing"

// TestLivePixels proves the pixel-present path a GTK4-hosted host needs: an RGBA
// frame becomes a GdkTexture, shows in a GtkPicture, and reads back correctly.
// gdk_texture_download always yields Cairo ARGB32 — BGRA, premultiplied — so for
// an opaque colour (premultiply is the identity) the R and B channels swap; the
// test checks exactly that mapping, which confirms the R8G8B8A8 input format was
// interpreted right.
func TestLivePixels(t *testing.T) {
	ok, err := Init()
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if !ok {
		t.Skip("no display; run under Xvfb")
	}

	const w, h = 4, 4
	src := make([]byte, w*h*4)
	for i := 0; i < w*h; i++ {
		src[i*4+0] = 10  // R
		src[i*4+1] = 20  // G
		src[i*4+2] = 30  // B
		src[i*4+3] = 255 // A (opaque)
	}

	tex := TextureFromRGBA(src, w, h)
	if tex == 0 {
		t.Fatal("TextureFromRGBA returned null — the pixel entry points did not load")
	}

	// Show it in a picture inside a window (proves set_paintable + the widget path).
	pic := PictureNew()
	if pic == 0 {
		t.Fatal("PictureNew returned null")
	}
	win := WindowNew()
	win.SetDefaultSize(64, 64)
	win.SetChild(pic.Widget())
	pic.SetRGBA(src, w, h)
	win.Present()

	// Read the texture back: expect BGRA (opaque, so premultiply is identity).
	dst := make([]byte, w*h*4)
	downloadRGBA(tex, dst, w)
	gotB, gotG, gotR, gotA := dst[0], dst[1], dst[2], dst[3]
	if gotB != 30 || gotG != 20 || gotR != 10 || gotA != 255 {
		t.Errorf("downloaded pixel = B%d G%d R%d A%d, want B30 G20 R10 A255 (RGBA 10,20,30 as Cairo BGRA)",
			gotB, gotG, gotR, gotA)
	}
	gObjectUnref(tex)
}
