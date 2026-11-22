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

// TestLiveUpdateRGBA proves the partial update shows the same pixels a whole
// SetRGBA would: frame A is set whole, frame B (A with one rectangle changed) is
// given through UpdateRGBA, and the picture reads back as B. Below GTK 4.16 the
// API is absent and UpdateRGBA must say so without touching the picture.
func TestLiveUpdateRGBA(t *testing.T) {
	ok, err := Init()
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if !ok {
		t.Skip("no display; run under Xvfb")
	}
	const w, h = 16, 16
	a := make([]byte, w*h*4)
	for i := 0; i < w*h; i++ {
		copy(a[i*4:], []byte{10, 20, 30, 255})
	}
	b := append([]byte(nil), a...)
	for y := 4; y < 8; y++ {
		for x := 6; x < 12; x++ {
			copy(b[(y*w+x)*4:], []byte{200, 100, 50, 255})
		}
	}
	pic := PictureNew()
	if pic == 0 {
		t.Fatal("PictureNew returned null")
	}
	rects := []Rect{{X: 6, Y: 4, W: 6, H: 4}}
	if pic.UpdateRGBA(b, w, h, rects) {
		t.Fatal("UpdateRGBA succeeded with no texture to update")
	}
	pic.SetRGBA(a, w, h)
	t.Logf("libgtk-4 minor %d, texture update supported: %v", MinorVersion(), TextureUpdateSupported())
	if !TextureUpdateSupported() {
		if os.Getenv("GTK4_REQUIRE_UPDATE") != "" {
			t.Fatalf("GTK4_REQUIRE_UPDATE is set but libgtk-4 4.%d lacks the update API", MinorVersion())
		}
		if pic.UpdateRGBA(b, w, h, rects) {
			t.Fatal("UpdateRGBA succeeded below GTK 4.16")
		}
		got := make([]byte, w*h*4)
		if !pic.ReadRGBA(got, w, h) || !bytes.Equal(got, a) {
			t.Fatal("a refused UpdateRGBA changed the picture")
		}
		return
	}
	if pic.UpdateRGBA(b, w+1, h, rects) || pic.UpdateRGBA(b, w, h, []Rect{{X: 100, Y: 100, W: 4, H: 4}}) {
		t.Fatal("UpdateRGBA accepted a wrong size or an off-frame rectangle")
	}
	if !pic.UpdateRGBA(b, w, h, rects) {
		t.Fatal("UpdateRGBA refused a valid update")
	}
	got := make([]byte, w*h*4)
	if !pic.ReadRGBA(got, w, h) {
		t.Fatal("ReadRGBA found no texture")
	}
	if !bytes.Equal(got, b) {
		t.Fatal("picture after UpdateRGBA differs from the whole frame")
	}
}
