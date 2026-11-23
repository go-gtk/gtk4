// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import (
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	frmOnce sync.Once
	frmOK   bool

	gMallocN       func(uint64, uint64) uintptr
	gBytesNewTake  func(uintptr, uint64) uintptr
	gtkOverlayNew  func() uintptr
	gtkOverlaySet  func(uintptr, uintptr)
	gtkOverlayAdd  func(uintptr, uintptr)
	gtkAllocWidth  func(uintptr) int32
	gtkAllocHeight func(uintptr) int32
)

func loadFrame() bool {
	frmOnce.Do(func() {
		if loadPixels() != nil {
			return
		}
		glib, err := dlopenAny("libglib-2.0.so.0", "libglib-2.0.so")
		if err != nil {
			return
		}
		gtk, err := dlopenAny("libgtk-4.so.1", "libgtk-4.so")
		if err != nil {
			return
		}
		purego.RegisterLibFunc(&gMallocN, glib, "g_malloc0_n")
		purego.RegisterLibFunc(&gBytesNewTake, glib, "g_bytes_new_take")
		purego.RegisterLibFunc(&gtkOverlayNew, gtk, "gtk_overlay_new")
		purego.RegisterLibFunc(&gtkOverlaySet, gtk, "gtk_overlay_set_child")
		purego.RegisterLibFunc(&gtkOverlayAdd, gtk, "gtk_overlay_add_overlay")
		purego.RegisterLibFunc(&gtkAllocWidth, gtk, "gtk_widget_get_allocated_width")
		purego.RegisterLibFunc(&gtkAllocHeight, gtk, "gtk_widget_get_allocated_height")
		frmOK = true
	})
	return frmOK
}

// A SharedFrame is a width*height R8G8B8A8 framebuffer that lives in GLib memory
// and is handed to GTK WITHOUT a copy: the texture [Picture.SetFrame] and
// [Picture.UpdateFrame] build refers to the very bytes [SharedFrame.Pix] returns,
// so a host that paints into Pix and reports the damaged rectangles pays no
// per-frame copy at all (SetRGBA and UpdateRGBA copy the whole frame, g_bytes_new).
//
// GdkMemoryTextureBuilder cannot take a partial buffer: gdk_memory_texture_builder_build
// refuses (g_return_val_if_fail, gdk/gdkmemorytexturebuilder.c:726 at 4.16.0) any
// GBytes smaller than gdk_memory_format_min_buffer_size(format, stride, width,
// height), so the new texture's bytes must span the whole frame even though
// update-region only tells GTK what it may skip re-rendering. Sharing the host's
// one buffer is how that requirement costs nothing.
//
// The texture sees later writes to Pix. That is exact for the partial path (a
// frame differs from the last only inside the reported region, which is what
// update-region promises) and shows the newest pixels if GTK re-reads an older
// texture. Pix is only valid until [SharedFrame.Release]; GTK keeps the memory
// alive for as long as a texture still refers to it.
type SharedFrame struct {
	bytes uintptr // GBytes owning the g_malloc'd pixels, one reference held here
	pix   []byte
	w, h  int
}

// NewSharedFrame allocates a zeroed width×height frame, or returns nil if the
// GTK libraries cannot be loaded or the size is not positive or overflows.
func NewSharedFrame(width, height int) *SharedFrame {
	if width <= 0 || height <= 0 || width > (1<<31-1)/4/height || !loadFrame() {
		return nil
	}
	n := uint64(width) * uint64(height) * 4
	mem := gMallocN(n, 1)
	if mem == 0 {
		return nil
	}
	b := gBytesNewTake(mem, n) // g_free'd when the last reference drops
	// The pointer is GLib heap, not Go heap, so the slice header may alias it.
	p := *(*unsafe.Pointer)(unsafe.Pointer(&mem))
	return &SharedFrame{bytes: b, pix: unsafe.Slice((*byte)(p), n), w: width, h: height}
}

// Pix is the frame's pixels, width*height*4 bytes of R8G8B8A8.
func (f *SharedFrame) Pix() []byte { return f.pix }

// Size is the frame's width and height in pixels.
func (f *SharedFrame) Size() (int, int) { return f.w, f.h }

// Release drops the host's reference. Pix must not be used afterwards; textures
// still showing the frame keep the memory alive until GTK is done with them.
func (f *SharedFrame) Release() {
	if f == nil || f.bytes == 0 {
		return
	}
	gBytesUnref(f.bytes)
	f.bytes, f.pix = 0, nil
}

// SetFrame shows the whole frame in the picture without copying it.
func (p Picture) SetFrame(f *SharedFrame) {
	if f == nil || f.bytes == 0 {
		return
	}
	tex := gdkMemoryTextureNew(int32(f.w), int32(f.h), MemoryR8G8B8A8, f.bytes, uint64(f.w*4))
	if tex == 0 {
		return
	}
	gtkPictureSetPaint(uintptr(p), tex)
	gObjectUnref(tex)
}

// UpdateFrame shows the frame after only rects changed, copying nothing: the
// counterpart of [Picture.UpdateRGBA] over a [SharedFrame]. It returns false,
// having changed nothing, in the cases UpdateRGBA does (GTK older than 4.16, no
// texture of exactly the frame's size shown yet, empty or off-frame rects).
func (p Picture) UpdateFrame(f *SharedFrame, rects []Rect) bool {
	if f == nil || f.bytes == 0 {
		return false
	}
	prev, region, ok := p.prepareUpdate(f.w, f.h, rects)
	if !ok {
		return false
	}
	return p.commitUpdate(f.bytes, f.w, f.h, prev, region)
}

// OverlayNew creates a GtkOverlay: one child fills it, and overlays sit above.
func OverlayNew() Widget {
	if !loadFrame() {
		return 0
	}
	return Widget(gtkOverlayNew())
}

// SetOverlayChild sets the widget the overlay is sized and allocated around.
func (w Widget) SetOverlayChild(child Widget) { gtkOverlaySet(uintptr(w), uintptr(child)) }

// AddOverlay puts child above the overlay's main child.
func (w Widget) AddOverlay(child Widget) { gtkOverlayAdd(uintptr(w), uintptr(child)) }

// AllocatedSize is the size GTK last gave the widget, in logical points, or 0, 0
// before the first layout (or if the library cannot be loaded).
func (w Widget) AllocatedSize() (int, int) {
	if !loadFrame() {
		return 0, 0
	}
	return int(gtkAllocWidth(uintptr(w))), int(gtkAllocHeight(uintptr(w)))
}
