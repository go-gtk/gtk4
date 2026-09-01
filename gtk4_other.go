// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build !(linux && !android && (amd64 || arm64))

// Away from Linux there is no GTK4 to bind. Every exported symbol the Linux build
// provides exists here too, so a consumer cross-compiles without a build tag of
// its own and finds out at run time — through [ErrUnsupported] from [Init] — that
// this platform has no GTK to drive. GTK4 does run on macOS and Windows, but a
// go-widgets host uses the platform's own toolkit there (AppKit, Win32); this
// package is the Linux path, so it stays Linux-only rather than pretending to be
// a portable GTK.

package gtk4

// Widget is a GTK widget. Off Linux no real one is ever created.
type Widget uintptr

// MainLoop is a GLib main loop. Off Linux it does nothing.
type MainLoop uintptr

// Init reports [ErrUnsupported] off Linux.
func Init() (bool, error) { return false, ErrUnsupported }

// The constructors return the null widget off Linux; the methods are no-ops or
// zero, so consumer code that names them still compiles.
func WindowNew() Widget                             { return 0 }
func (w Widget) SetTitle(string)                    {}
func (w Widget) SetDefaultSize(int, int)            {}
func (w Widget) SetChild(Widget)                    {}
func (w Widget) Present()                           {}
func FixedNew() Widget                              { return 0 }
func (f Widget) Put(Widget, float64, float64)       {}
func (f Widget) Move(Widget, float64, float64)      {}
func ButtonNewWithLabel(string) Widget              { return 0 }
func (w Widget) SetLabel(string)                    {}
func CheckButtonNewWithLabel(string) Widget         { return 0 }
func (w Widget) Active() bool                       { return false }
func (w Widget) SetActive(bool)                     {}
func EntryNew() Widget                              { return 0 }
func (w Widget) SetVisibility(bool)                 {}
func LabelNew(string) Widget                        { return 0 }
func SliderNew(float64, float64, float64) Widget    { return 0 }
func (w Widget) Value() float64                     { return 0 }
func (w Widget) SetValue(float64)                   {}
func PopUpNew([]string) Widget                      { return 0 }
func (w Widget) Selected() int                      { return -1 }
func (w Widget) SetSelected(int)                    {}
func (w Widget) Text() string                       { return "" }
func (w Widget) SetText(string)                     {}
func (w Widget) SetSizeRequest(int, int)            {}
func (w Widget) SetVisible(bool)                    {}
func (w Widget) Unparent()                          {}
func (w Widget) Connect(string, func()) uint64      { return 0 }
func (w Widget) AddTickCallback(func() bool) uint64 { return 0 }
func MainLoopNew() MainLoop                         { return 0 }
func (l MainLoop) Run()                             {}
func (l MainLoop) Quit()                            {}
func IdleAdd(func())                                {}

// MemoryR8G8B8A8 is the GdkMemoryFormat for a go-widgets RGBA frame; defined here
// too so consumer code naming it compiles off Linux.
const MemoryR8G8B8A8 = 5

// Picture is a GtkPicture showing an RGBA frame. Off Linux it does nothing.
type Picture uintptr

func PictureNew() Picture                      { return 0 }
func (p Picture) Widget() Widget               { return 0 }
func TextureFromRGBA([]byte, int, int) uintptr { return 0 }
func (p Picture) SetRGBA([]byte, int, int)     {}
func DrawingAreaNew() Widget                   { return 0 }
func (w Widget) QueueDraw()                    {}

// GDK modifier-state bits, defined here too so consumer code naming them compiles
// off Linux.
const (
	ModShift   uint = 1 << 0
	ModControl uint = 1 << 2
	ModAlt     uint = 1 << 3
	ModSuper   uint = 1 << 26
)

// The input event controllers do nothing off Linux; a host there uses the
// platform's own toolkit for input.
func KeyvalToUnicode(uint) rune                                { return 0 }
func (w Widget) OnMouseDown(func(int, uint, float64, float64)) {}
func (w Widget) OnMouseUp(func(int, uint, float64, float64))   {}
func (w Widget) OnMotion(func(uint, float64, float64))         {}
func (w Widget) OnScroll(func(float64, float64, uint))         {}
func (w Widget) OnKey(func(uint, uint, uint, bool))            {}
