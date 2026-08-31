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

// MemoryR8G8B8A8 is GdkMemoryFormat GDK_MEMORY_R8G8B8A8: four bytes per pixel in
// R, G, B, A order, not premultiplied — the layout a go-widgets pixel buffer
// uses, so a frame maps to a GdkTexture with no conversion.
const MemoryR8G8B8A8 = 5

var (
	pixOnce sync.Once
	pixErr  error

	gBytesNew           func(uintptr, uint64) uintptr
	gBytesUnref         func(uintptr)
	gObjectUnref        func(uintptr)
	gdkMemoryTextureNew func(int32, int32, uint32, uintptr, uint64) uintptr
	gdkTextureDownload  func(uintptr, uintptr, uint64)
	gtkPictureNew       func() uintptr
	gtkPictureSetPaint  func(uintptr, uintptr)
	gtkDrawingAreaNew   func() uintptr
	gtkWidgetQueueDraw  func(uintptr)
)

// loadPixels resolves the texture/picture entry points. The libraries are the
// same the core load() opened; dlopen is cached, so opening them again is cheap.
func loadPixels() error {
	pixOnce.Do(func() {
		glib, err := dlopenAny("libglib-2.0.so.0", "libglib-2.0.so")
		if err != nil {
			pixErr = err
			return
		}
		gobj, err := dlopenAny("libgobject-2.0.so.0", "libgobject-2.0.so")
		if err != nil {
			pixErr = err
			return
		}
		gtk, err := dlopenAny("libgtk-4.so.1", "libgtk-4.so")
		if err != nil {
			pixErr = err
			return
		}
		purego.RegisterLibFunc(&gBytesNew, glib, "g_bytes_new")
		purego.RegisterLibFunc(&gBytesUnref, glib, "g_bytes_unref")
		purego.RegisterLibFunc(&gObjectUnref, gobj, "g_object_unref")
		purego.RegisterLibFunc(&gdkMemoryTextureNew, gtk, "gdk_memory_texture_new")
		purego.RegisterLibFunc(&gdkTextureDownload, gtk, "gdk_texture_download")
		purego.RegisterLibFunc(&gtkPictureNew, gtk, "gtk_picture_new")
		purego.RegisterLibFunc(&gtkPictureSetPaint, gtk, "gtk_picture_set_paintable")
		purego.RegisterLibFunc(&gtkDrawingAreaNew, gtk, "gtk_drawing_area_new")
		purego.RegisterLibFunc(&gtkWidgetQueueDraw, gtk, "gtk_widget_queue_draw")
	})
	return pixErr
}

// A Picture is a GtkPicture that shows an RGBA image a host refreshes each frame
// with [Picture.SetRGBA] — the widget the GTK4-hosted go-widgets backend puts the
// toolkit's pixel framebuffer into, with native controls overlaid above it.
type Picture Widget

// PictureNew creates a picture widget. It returns the null Picture if the pixel
// entry points cannot be loaded.
func PictureNew() Picture {
	if err := loadPixels(); err != nil {
		return 0
	}
	return Picture(gtkPictureNew())
}

// Widget is the picture as a plain widget, for putting into a container.
func (p Picture) Widget() Widget { return Widget(p) }

// TextureFromRGBA builds a GdkTexture from width*height*4 bytes of R8G8B8A8. The
// bytes are copied (g_bytes_new copies), so the caller may reuse its buffer at
// once. The returned handle is a GdkTexture with one reference the caller owns.
func TextureFromRGBA(data []byte, width, height int) uintptr {
	if err := loadPixels(); err != nil || len(data) < width*height*4 {
		return 0
	}
	b := gBytesNew(uintptr(unsafe.Pointer(&data[0])), uint64(width*height*4))
	tex := gdkMemoryTextureNew(int32(width), int32(height), MemoryR8G8B8A8, b, uint64(width*4))
	gBytesUnref(b) // the texture holds its own reference to the copied bytes
	return tex
}

// SetRGBA shows width*height*4 bytes of R8G8B8A8 in the picture. It is safe to
// call every frame: a fresh texture is set and the previous one released (the
// picture took a reference, so unref-ing the caller's is correct).
func (p Picture) SetRGBA(data []byte, width, height int) {
	tex := TextureFromRGBA(data, width, height)
	if tex == 0 {
		return
	}
	gtkPictureSetPaint(uintptr(p), tex)
	gObjectUnref(tex) // the picture now holds the only reference
}

// DrawingAreaNew creates a GtkDrawingArea. QueueDraw asks a widget to repaint.
func DrawingAreaNew() Widget { _ = loadPixels(); return Widget(gtkDrawingAreaNew()) }
func (w Widget) QueueDraw()  { gtkWidgetQueueDraw(uintptr(w)) }

// downloadRGBA reads a GdkTexture's pixels back into dst (width*height*4,
// R8G8B8A8), for tests that verify the pixel path round-trips.
func downloadRGBA(tex uintptr, dst []byte, width int) {
	gdkTextureDownload(tex, uintptr(unsafe.Pointer(&dst[0])), uint64(width*4))
}
