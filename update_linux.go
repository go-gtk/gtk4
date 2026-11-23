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

// Rect is a pixel rectangle: the damaged part of a frame.
type Rect struct{ X, Y, W, H int }

// minTextureUpdateMinor is the first GTK 4 minor release with
// GdkMemoryTextureBuilder (and its update-texture / update-region properties):
// "Available since: 4.16", https://docs.gtk.org/gdk4/class.MemoryTextureBuilder.html
const minTextureUpdateMinor = 16

var (
	updOnce sync.Once
	updOK   bool
	minor   int

	gtkGetMinorVersion func() uint32

	gtkPictureGetPaintable func(uintptr) uintptr
	gdkTextureWidth        func(uintptr) int32
	gdkTextureHeight       func(uintptr) int32

	mtbNew       func() uintptr
	mtbSetBytes  func(uintptr, uintptr)
	mtbSetWidth  func(uintptr, int32)
	mtbSetHeight func(uintptr, int32)
	mtbSetFormat func(uintptr, uint32)
	mtbSetStride func(uintptr, uint64)
	mtbSetUpdTex func(uintptr, uintptr)
	mtbSetUpdReg func(uintptr, uintptr)
	mtbBuild     func(uintptr) uintptr

	cairoRegionCreate   func() uintptr
	cairoRegionUnionRec func(uintptr, uintptr) int32
	cairoRegionDestroy  func(uintptr)
)

// cairoRectInt is cairo_rectangle_int_t.
type cairoRectInt struct{ x, y, w, h int32 }

func loadUpdate() {
	updOnce.Do(func() {
		if loadPixels() != nil {
			return
		}
		gtk, err := dlopenAny("libgtk-4.so.1", "libgtk-4.so")
		if err != nil {
			return
		}
		// GTK 4.0 symbols, needed by ReadRGBA on any runtime.
		purego.RegisterLibFunc(&gtkPictureGetPaintable, gtk, "gtk_picture_get_paintable")
		purego.RegisterLibFunc(&gdkTextureWidth, gtk, "gdk_texture_get_width")
		purego.RegisterLibFunc(&gdkTextureHeight, gtk, "gdk_texture_get_height")
		purego.RegisterLibFunc(&gtkGetMinorVersion, gtk, "gtk_get_minor_version")
		minor = int(gtkGetMinorVersion())
		if minor < minTextureUpdateMinor {
			return
		}
		cairo, err := dlopenAny("libcairo.so.2", "libcairo.so")
		if err != nil {
			return
		}
		purego.RegisterLibFunc(&mtbNew, gtk, "gdk_memory_texture_builder_new")
		purego.RegisterLibFunc(&mtbSetBytes, gtk, "gdk_memory_texture_builder_set_bytes")
		purego.RegisterLibFunc(&mtbSetWidth, gtk, "gdk_memory_texture_builder_set_width")
		purego.RegisterLibFunc(&mtbSetHeight, gtk, "gdk_memory_texture_builder_set_height")
		purego.RegisterLibFunc(&mtbSetFormat, gtk, "gdk_memory_texture_builder_set_format")
		purego.RegisterLibFunc(&mtbSetStride, gtk, "gdk_memory_texture_builder_set_stride")
		purego.RegisterLibFunc(&mtbSetUpdTex, gtk, "gdk_memory_texture_builder_set_update_texture")
		purego.RegisterLibFunc(&mtbSetUpdReg, gtk, "gdk_memory_texture_builder_set_update_region")
		purego.RegisterLibFunc(&mtbBuild, gtk, "gdk_memory_texture_builder_build")
		purego.RegisterLibFunc(&cairoRegionCreate, cairo, "cairo_region_create")
		purego.RegisterLibFunc(&cairoRegionUnionRec, cairo, "cairo_region_union_rectangle")
		purego.RegisterLibFunc(&cairoRegionDestroy, cairo, "cairo_region_destroy")
		updOK = true
	})
}

// MinorVersion is the minor version of the libgtk-4 loaded at run time
// (gtk_get_minor_version), or 0 if it cannot be loaded.
func MinorVersion() int {
	loadUpdate()
	return minor
}

// TextureUpdateSupported reports whether the loaded libgtk-4 has
// GdkMemoryTextureBuilder (GTK 4.16 or newer), the API [Picture.UpdateRGBA]
// needs. When false, callers keep to [Picture.SetRGBA].
func TextureUpdateSupported() bool {
	loadUpdate()
	return updOK
}

// UpdateRGBA replaces the picture's texture with the width*height*4 bytes of
// R8G8B8A8 in data, telling GTK that only rects differ from the texture the
// picture shows now (GdkMemoryTextureBuilder:update-texture and
// :update-region), so the renderer can avoid re-rendering the rest. data is
// still the WHOLE frame and is copied whole: only GTK's own re-render and
// texture update is narrowed.
//
// It returns false, having changed nothing, when the update API is missing
// (GTK older than 4.16), when the picture shows no texture of exactly
// width×height yet (the first frame, a resize), or when rects is empty or lies
// outside the frame; the caller then calls [Picture.SetRGBA].
func (p Picture) UpdateRGBA(data []byte, width, height int, rects []Rect) bool {
	if width <= 0 || height <= 0 || len(data) < width*height*4 {
		return false
	}
	prev, region, ok := p.prepareUpdate(width, height, rects)
	if !ok {
		return false
	}
	b := gBytesNew(uintptr(unsafe.Pointer(&data[0])), uint64(width*height*4))
	done := p.commitUpdate(b, width, height, prev, region)
	gBytesUnref(b)
	return done
}

// prepareUpdate checks that a partial update is possible and returns the texture
// the picture shows and the cairo region of rects, which the caller owns (and
// commitUpdate consumes). ok is false, with nothing allocated, when it is not.
func (p Picture) prepareUpdate(width, height int, rects []Rect) (prev, region uintptr, ok bool) {
	if !TextureUpdateSupported() || width <= 0 || height <= 0 {
		return 0, 0, false
	}
	prev = gtkPictureGetPaintable(uintptr(p))
	if prev == 0 || int(gdkTextureWidth(prev)) != width || int(gdkTextureHeight(prev)) != height {
		return 0, 0, false
	}
	region = cairoRegionCreate()
	n := 0
	for _, r := range rects {
		x0, y0, x1, y1 := max(r.X, 0), max(r.Y, 0), min(r.X+r.W, width), min(r.Y+r.H, height)
		if x1 <= x0 || y1 <= y0 {
			continue
		}
		cr := cairoRectInt{int32(x0), int32(y0), int32(x1 - x0), int32(y1 - y0)}
		cairoRegionUnionRec(region, uintptr(unsafe.Pointer(&cr)))
		n++
	}
	if n == 0 {
		cairoRegionDestroy(region)
		return 0, 0, false
	}
	return prev, region, true
}

// commitUpdate builds the texture over bytes b (which must span the whole frame)
// as an update of prev limited to region, sets it, and destroys region. b's
// reference stays the caller's.
func (p Picture) commitUpdate(b uintptr, width, height int, prev, region uintptr) bool {
	bld := mtbNew()
	mtbSetBytes(bld, b)
	mtbSetWidth(bld, int32(width))
	mtbSetHeight(bld, int32(height))
	mtbSetFormat(bld, MemoryR8G8B8A8)
	mtbSetStride(bld, uint64(width*4))
	mtbSetUpdTex(bld, prev)
	mtbSetUpdReg(bld, region)
	tex := mtbBuild(bld)
	gObjectUnref(bld)
	cairoRegionDestroy(region)
	if tex == 0 {
		return false
	}
	gtkPictureSetPaint(uintptr(p), tex)
	gObjectUnref(tex)
	return true
}

// ReadRGBA copies the pixels the picture shows now into dst (width*height*4,
// R8G8B8A8), for tests that compare what is on screen with what was drawn.
// GDK downloads Cairo ARGB32 (BGRA, premultiplied); the R and B channels are
// swapped back, exact for opaque pixels. It returns false when the picture
// shows no texture of that size.
func (p Picture) ReadRGBA(dst []byte, width, height int) bool {
	if loadPixels() != nil || width <= 0 || height <= 0 || len(dst) < width*height*4 {
		return false
	}
	loadUpdate()
	if gtkPictureGetPaintable == nil {
		return false
	}
	tex := gtkPictureGetPaintable(uintptr(p))
	if tex == 0 || int(gdkTextureWidth(tex)) != width || int(gdkTextureHeight(tex)) != height {
		return false
	}
	downloadRGBA(tex, dst, width)
	for i := 0; i+3 < width*height*4; i += 4 {
		dst[i], dst[i+2] = dst[i+2], dst[i]
	}
	return true
}
