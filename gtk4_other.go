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
func ListBoxNew() Widget                            { return 0 }
func (w Widget) ListBoxAppendText(string)           {}
func (w Widget) SelectedRow() int                   { return -1 }
func (w Widget) SelectRow(int)                      {}
func (w Widget) Text() string                       { return "" }
func (w Widget) SetText(string)                     {}
func (w Widget) SetSizeRequest(int, int)            {}
func (w Widget) SetVisible(bool)                    {}
func (w Widget) Unparent()                          {}
func (w Widget) SetOverflowHidden(bool)             {}
func (w Widget) Parent() Widget                     { return 0 }
func (w Widget) Connect(string, func()) uint64      { return 0 }
func (w Widget) AddTickCallback(func() bool) uint64 { return 0 }

// Widgets added for the go-widgets host backend; null/no-op/zero off Linux.
func ProgressNew() Widget                           { return 0 }
func (w Widget) SetFraction(float64)                {}
func SpinnerNew() Widget                            { return 0 }
func (w Widget) Start()                             {}
func (w Widget) Stop()                              {}
func StepperNew(float64, float64, float64) Widget   { return 0 }
func (w Widget) SpinValue() float64                 { return 0 }
func (w Widget) SetSpinValue(float64)               {}
func SearchNew() Widget                             { return 0 }
func ComboNew([]string) Widget                      { return 0 }
func (w Widget) ComboText() string                  { return "" }
func (w Widget) SetComboText(string)                {}
func BoxNew(bool) Widget                            { return 0 }
func (w Widget) Append(Widget)                      {}
func ToggleButtonNewWithLabel(string) Widget        { return 0 }
func (w Widget) SetGroup(Widget)                    {}
func TextViewNew() Widget                           { return 0 }
func (w Widget) Buffer() Widget                     { return 0 }
func (w Widget) ConnectBufferChanged(func()) uint64 { return 0 }
func (w Widget) SetTextViewText(string)             {}
func (w Widget) TextViewText() string               { return "" }
func LinkNew(string) Widget                         { return 0 }
func (w Widget) OnActivateLink(func()) uint64       { return 0 }
func DateNew() Widget                               { return 0 }
func (w Widget) DateISO() string                    { return "" }
func (w Widget) SetDateISO(string)                  {}
func ColorNew() Widget                              { return 0 }
func (w Widget) ColorHex() string                   { return "" }
func (w Widget) SetColorHex(string)                 {}
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

// Rect is a pixel rectangle: the damaged part of a frame.
type Rect struct{ X, Y, W, H int }

func MinorVersion() int                                    { return 0 }
func TextureUpdateSupported() bool                         { return false }
func (p Picture) UpdateRGBA([]byte, int, int, []Rect) bool { return false }
func (p Picture) ReadRGBA([]byte, int, int) bool           { return false }
func (w Widget) QueueDraw()                                {}

// SharedFrame is a zero-copy framebuffer shared with GTK. Off Linux none exists.
type SharedFrame struct{}

func NewSharedFrame(int, int) *SharedFrame              { return nil }
func (f *SharedFrame) Pix() []byte                      { return nil }
func (f *SharedFrame) Size() (int, int)                 { return 0, 0 }
func (f *SharedFrame) Release()                         {}
func (p Picture) SetFrame(*SharedFrame)                 {}
func (p Picture) UpdateFrame(*SharedFrame, []Rect) bool { return false }
func OverlayNew() Widget                                { return 0 }
func (w Widget) SetOverlayChild(Widget)                 {}
func (w Widget) AddOverlay(Widget)                      {}
func (w Widget) AllocatedSize() (int, int)              { return 0, 0 }

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

// ContextMenu is a GtkPopoverMenu over a GMenu. Off Linux none is ever created.
type ContextMenu struct{}

func ContextMenuNew(Widget) *ContextMenu      { return nil }
func (m *ContextMenu) SetItems([]MenuItem)    {}
func (m *ContextMenu) Popup(float64, float64) {}
func (m *ContextMenu) Popdown()               {}
func (m *ContextMenu) IsOpen() bool           { return false }
func (m *ContextMenu) Len() int               { return 0 }
func (m *ContextMenu) Activate(int) bool      { return false }
func (m *ContextMenu) Destroy()               {}

func (w Widget) OnCloseRequest(func() bool) {}
func (w Widget) SetTooltip(string)          {}
func (w Widget) Close()                     {}
func (w Widget) Mapped() bool               { return false }
func (w Widget) SetButtonChild(Widget)      {}
func (p Picture) SetCanShrink(bool)         {}
func (p Picture) SetAlternativeText(string) {}

// IMContext is a GtkIMContext. Off Linux no real one is ever created.
type IMContext uintptr

func IMContextNew() IMContext                                    { return 0 }
func IMContextSimpleNew() IMContext                              { return 0 }
func (c IMContext) SetClientWidget(Widget)                       {}
func (c IMContext) FocusIn()                                     {}
func (c IMContext) FocusOut()                                    {}
func (c IMContext) Reset()                                       {}
func (c IMContext) SetCursorLocation(int, int, int, int)         {}
func (c IMContext) Preedit() (string, int)                       { return "", 0 }
func (c IMContext) OnPreeditChanged(func())                      {}
func (c IMContext) OnCommit(func(string))                        {}
func (c IMContext) Free()                                        {}
func (w Widget) OnKeyIM(IMContext, func(uint, uint, uint, bool)) {}

// MenuBar is the application menu bar; off Linux there is none.
type MenuBar struct{}

func (w Widget) Prepend(Widget)       {}
func (w Widget) SetExpand(h, v bool)  {}
func MenuBarSupported() bool          { return false }
func MenuBarNew(Widget) *MenuBar      { return nil }
func (m *MenuBar) Widget() Widget     { return 0 }
func (m *MenuBar) Len() int           { return 0 }
func (m *MenuBar) SetItems([]BarItem) {}
func (m *MenuBar) Activate(int) bool  { return false }
func (m *MenuBar) Destroy()           {}
