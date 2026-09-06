// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

// Package gtk4 binds the parts of GTK4 a windowing toolkit needs — windows, the
// common controls, GObject signals and the GLib main loop — from pure Go with
// CGO_ENABLED=0, over github.com/ebitengine/purego. It is the Linux sibling of
// github.com/go-macos/objc: the shared native-toolkit bridge a go-widgets host
// backend embeds real controls through.
//
// It links no cgo: libgtk-4, libgobject-2.0 and libglib-2.0 are dlopen'd by
// soname and every entry point is resolved with purego.RegisterLibFunc, exactly
// as the feasibility spike proved on a stock Linux GTK4. A GObject signal reaches
// Go through a purego.NewCallback, so a control's clicks and edits are ordinary
// Go closures.
package gtk4

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// A Widget is a GTK widget — a GObject pointer. The zero value is the null
// widget. Methods on it are thin wrappers over the C API.
type Widget uintptr

// A MainLoop is a GLib main loop.
type MainLoop uintptr

var (
	loadOnce sync.Once
	loadErr  error

	gtkInitCheck            func() bool
	gtkWindowNew            func() uintptr
	gtkWindowSetTitle       func(uintptr, string)
	gtkWindowSetDefault     func(uintptr, int32, int32)
	gtkWindowSetChild       func(uintptr, uintptr)
	gtkWindowPresent        func(uintptr)
	gtkFixedNew             func() uintptr
	gtkFixedPut             func(uintptr, uintptr, float64, float64)
	gtkFixedMove            func(uintptr, uintptr, float64, float64)
	gtkButtonNewWithLabel   func(string) uintptr
	gtkButtonSetLabel       func(uintptr, string)
	gtkCheckButtonNewWith   func(string) uintptr
	gtkCheckButtonActive    func(uintptr) bool
	gtkCheckButtonSetActive func(uintptr, bool)
	gtkEntryNew             func() uintptr
	gtkEntrySetVisibility   func(uintptr, bool)
	gtkLabelNew             func(string) uintptr
	gtkEditableGetText      func(uintptr) string
	gtkEditableSetText      func(uintptr, string)
	gtkWidgetSetSizeReq     func(uintptr, int32, int32)
	gtkWidgetSetVisible     func(uintptr, bool)
	gtkWidgetUnparent       func(uintptr)

	gSignalConnectData func(uintptr, string, uintptr, uintptr, uintptr, int32) uint64
	gMainLoopNew       func(uintptr, bool) uintptr
	gMainLoopRun       func(uintptr)
	gMainLoopQuit      func(uintptr)
	gIdleAdd           func(uintptr, uintptr) uint32
	gtkWidgetAddTick   func(uintptr, uintptr, uintptr, uintptr) uint32

	gtkScaleNewWithRange func(int32, float64, float64, float64) uintptr
	gtkRangeGetValue     func(uintptr) float64
	gtkRangeSetValue     func(uintptr, float64)
	gtkStringListNew     func(uintptr) uintptr
	gtkStringListAppend  func(uintptr, string)
	gtkDropDownNew       func(uintptr, uintptr) uintptr
	gtkDropDownGetSel    func(uintptr) uint32
	gtkDropDownSetSel    func(uintptr, uint32)

	gtkListBoxNew            func() uintptr
	gtkListBoxAppend         func(uintptr, uintptr)
	gtkListBoxGetSelectedRow func(uintptr) uintptr
	gtkListBoxRowGetIndex    func(uintptr) int32
	gtkListBoxGetRowAtIndex  func(uintptr, int32) uintptr
	gtkListBoxSelectRow      func(uintptr, uintptr)

	gtkWidgetAddController func(uintptr, uintptr)
	gtkGestureClickNew     func() uintptr
	gtkGestureSingleSetBtn func(uintptr, uint32)
	gtkGestureSingleGetBtn func(uintptr) uint32
	gtkEventCtrlGetState   func(uintptr) uint32
	gtkEventCtrlMotionNew  func() uintptr
	gtkEventCtrlScrollNew  func(uint32) uintptr
	gtkEventCtrlKeyNew     func() uintptr
	gdkKeyvalToUnicode     func(uint32) uint32

	// Widgets added for the go-widgets host backend (progress, spinner, spin
	// button, search entry, editable combo, box/toggle primitives, text view,
	// link, calendar, colour button).
	gtkProgressBarNew         func() uintptr
	gtkProgressBarSetFraction func(uintptr, float64)
	gtkSpinnerNew             func() uintptr
	gtkSpinnerStart           func(uintptr)
	gtkSpinnerStop            func(uintptr)
	gtkSpinButtonNewWithRange func(float64, float64, float64) uintptr
	gtkSpinButtonGetValue     func(uintptr) float64
	gtkSpinButtonSetValue     func(uintptr, float64)
	gtkSearchEntryNew         func() uintptr
	gtkComboBoxTextNewEntry   func() uintptr
	gtkComboBoxTextAppend     func(uintptr, string)
	gtkComboBoxTextActiveText func(uintptr) string
	gtkComboBoxGetChild       func(uintptr) uintptr
	gtkBoxNew                 func(int32, int32) uintptr
	gtkBoxAppend              func(uintptr, uintptr)
	gtkToggleButtonNewLabel   func(string) uintptr
	gtkToggleButtonSetGroup   func(uintptr, uintptr)
	gtkToggleButtonGetActive  func(uintptr) bool
	gtkToggleButtonSetActive  func(uintptr, bool)
	gtkToggleButtonGetType    func() uint64
	gTypeCheckInstanceIsA     func(uintptr, uint64) bool
	gtkTextViewNew            func() uintptr
	gtkTextViewGetBuffer      func(uintptr) uintptr
	gtkTextBufferSetText      func(uintptr, string, int32)
	gtkTextBufferGetStartIter func(uintptr, uintptr)
	gtkTextBufferGetEndIter   func(uintptr, uintptr)
	gtkTextBufferGetText      func(uintptr, uintptr, uintptr, bool) string
	gtkLinkButtonNewLabel     func(string, string) uintptr
	gtkCalendarNew            func() uintptr
	gtkCalendarGetDate        func(uintptr) uintptr
	gtkCalendarSelectDay      func(uintptr, uintptr)
	gDateTimeNewLocal         func(int32, int32, int32, int32, int32, float64) uintptr
	gDateTimeGetYear          func(uintptr) int32
	gDateTimeGetMonth         func(uintptr) int32
	gDateTimeGetDayOfMonth    func(uintptr) int32
	gDateTimeUnref            func(uintptr)
	gtkColorButtonNew         func() uintptr
	gtkColorChooserGetRGBA    func(uintptr, uintptr)
	gtkColorChooserSetRGBA    func(uintptr, uintptr)
)

func load() error {
	loadOnce.Do(func() {
		glib, err := dlopenAny("libglib-2.0.so.0", "libglib-2.0.so")
		if err != nil {
			loadErr = err
			return
		}
		gobj, err := dlopenAny("libgobject-2.0.so.0", "libgobject-2.0.so")
		if err != nil {
			loadErr = err
			return
		}
		gtk, err := dlopenAny("libgtk-4.so.1", "libgtk-4.so")
		if err != nil {
			loadErr = err
			return
		}
		reg := func(p any, h uintptr, name string) { purego.RegisterLibFunc(p, h, name) }
		reg(&gtkInitCheck, gtk, "gtk_init_check")
		reg(&gtkWindowNew, gtk, "gtk_window_new")
		reg(&gtkWindowSetTitle, gtk, "gtk_window_set_title")
		reg(&gtkWindowSetDefault, gtk, "gtk_window_set_default_size")
		reg(&gtkWindowSetChild, gtk, "gtk_window_set_child")
		reg(&gtkWindowPresent, gtk, "gtk_window_present")
		reg(&gtkFixedNew, gtk, "gtk_fixed_new")
		reg(&gtkFixedPut, gtk, "gtk_fixed_put")
		reg(&gtkFixedMove, gtk, "gtk_fixed_move")
		reg(&gtkButtonNewWithLabel, gtk, "gtk_button_new_with_label")
		reg(&gtkButtonSetLabel, gtk, "gtk_button_set_label")
		reg(&gtkCheckButtonNewWith, gtk, "gtk_check_button_new_with_label")
		reg(&gtkCheckButtonActive, gtk, "gtk_check_button_get_active")
		reg(&gtkCheckButtonSetActive, gtk, "gtk_check_button_set_active")
		reg(&gtkEntryNew, gtk, "gtk_entry_new")
		reg(&gtkEntrySetVisibility, gtk, "gtk_entry_set_visibility")
		reg(&gtkLabelNew, gtk, "gtk_label_new")
		reg(&gtkEditableGetText, gtk, "gtk_editable_get_text")
		reg(&gtkEditableSetText, gtk, "gtk_editable_set_text")
		reg(&gtkWidgetSetSizeReq, gtk, "gtk_widget_set_size_request")
		reg(&gtkWidgetSetVisible, gtk, "gtk_widget_set_visible")
		reg(&gtkWidgetUnparent, gtk, "gtk_widget_unparent")
		reg(&gSignalConnectData, gobj, "g_signal_connect_data")
		reg(&gMainLoopNew, glib, "g_main_loop_new")
		reg(&gMainLoopRun, glib, "g_main_loop_run")
		reg(&gMainLoopQuit, glib, "g_main_loop_quit")
		reg(&gIdleAdd, glib, "g_idle_add")
		reg(&gtkWidgetAddTick, gtk, "gtk_widget_add_tick_callback")
		reg(&gtkScaleNewWithRange, gtk, "gtk_scale_new_with_range")
		reg(&gtkRangeGetValue, gtk, "gtk_range_get_value")
		reg(&gtkRangeSetValue, gtk, "gtk_range_set_value")
		reg(&gtkStringListNew, gtk, "gtk_string_list_new")
		reg(&gtkStringListAppend, gtk, "gtk_string_list_append")
		reg(&gtkDropDownNew, gtk, "gtk_drop_down_new")
		reg(&gtkDropDownGetSel, gtk, "gtk_drop_down_get_selected")
		reg(&gtkDropDownSetSel, gtk, "gtk_drop_down_set_selected")
		reg(&gtkListBoxNew, gtk, "gtk_list_box_new")
		reg(&gtkListBoxAppend, gtk, "gtk_list_box_append")
		reg(&gtkListBoxGetSelectedRow, gtk, "gtk_list_box_get_selected_row")
		reg(&gtkListBoxRowGetIndex, gtk, "gtk_list_box_row_get_index")
		reg(&gtkListBoxGetRowAtIndex, gtk, "gtk_list_box_get_row_at_index")
		reg(&gtkListBoxSelectRow, gtk, "gtk_list_box_select_row")
		reg(&gtkWidgetAddController, gtk, "gtk_widget_add_controller")
		reg(&gtkGestureClickNew, gtk, "gtk_gesture_click_new")
		reg(&gtkGestureSingleSetBtn, gtk, "gtk_gesture_single_set_button")
		reg(&gtkGestureSingleGetBtn, gtk, "gtk_gesture_single_get_current_button")
		reg(&gtkEventCtrlGetState, gtk, "gtk_event_controller_get_current_event_state")
		reg(&gtkEventCtrlMotionNew, gtk, "gtk_event_controller_motion_new")
		reg(&gtkEventCtrlScrollNew, gtk, "gtk_event_controller_scroll_new")
		reg(&gtkEventCtrlKeyNew, gtk, "gtk_event_controller_key_new")
		reg(&gdkKeyvalToUnicode, gtk, "gdk_keyval_to_unicode")

		reg(&gtkProgressBarNew, gtk, "gtk_progress_bar_new")
		reg(&gtkProgressBarSetFraction, gtk, "gtk_progress_bar_set_fraction")
		reg(&gtkSpinnerNew, gtk, "gtk_spinner_new")
		reg(&gtkSpinnerStart, gtk, "gtk_spinner_start")
		reg(&gtkSpinnerStop, gtk, "gtk_spinner_stop")
		reg(&gtkSpinButtonNewWithRange, gtk, "gtk_spin_button_new_with_range")
		reg(&gtkSpinButtonGetValue, gtk, "gtk_spin_button_get_value")
		reg(&gtkSpinButtonSetValue, gtk, "gtk_spin_button_set_value")
		reg(&gtkSearchEntryNew, gtk, "gtk_search_entry_new")
		reg(&gtkComboBoxTextNewEntry, gtk, "gtk_combo_box_text_new_with_entry")
		reg(&gtkComboBoxTextAppend, gtk, "gtk_combo_box_text_append_text")
		reg(&gtkComboBoxTextActiveText, gtk, "gtk_combo_box_text_get_active_text")
		reg(&gtkComboBoxGetChild, gtk, "gtk_combo_box_get_child")
		reg(&gtkBoxNew, gtk, "gtk_box_new")
		reg(&gtkBoxAppend, gtk, "gtk_box_append")
		reg(&gtkToggleButtonNewLabel, gtk, "gtk_toggle_button_new_with_label")
		reg(&gtkToggleButtonSetGroup, gtk, "gtk_toggle_button_set_group")
		reg(&gtkToggleButtonGetActive, gtk, "gtk_toggle_button_get_active")
		reg(&gtkToggleButtonSetActive, gtk, "gtk_toggle_button_set_active")
		reg(&gtkToggleButtonGetType, gtk, "gtk_toggle_button_get_type")
		reg(&gTypeCheckInstanceIsA, gobj, "g_type_check_instance_is_a")
		reg(&gtkTextViewNew, gtk, "gtk_text_view_new")
		reg(&gtkTextViewGetBuffer, gtk, "gtk_text_view_get_buffer")
		reg(&gtkTextBufferSetText, gtk, "gtk_text_buffer_set_text")
		reg(&gtkTextBufferGetStartIter, gtk, "gtk_text_buffer_get_start_iter")
		reg(&gtkTextBufferGetEndIter, gtk, "gtk_text_buffer_get_end_iter")
		reg(&gtkTextBufferGetText, gtk, "gtk_text_buffer_get_text")
		reg(&gtkLinkButtonNewLabel, gtk, "gtk_link_button_new_with_label")
		reg(&gtkCalendarNew, gtk, "gtk_calendar_new")
		reg(&gtkCalendarGetDate, gtk, "gtk_calendar_get_date")
		reg(&gtkCalendarSelectDay, gtk, "gtk_calendar_select_day")
		reg(&gDateTimeNewLocal, glib, "g_date_time_new_local")
		reg(&gDateTimeGetYear, glib, "g_date_time_get_year")
		reg(&gDateTimeGetMonth, glib, "g_date_time_get_month")
		reg(&gDateTimeGetDayOfMonth, glib, "g_date_time_get_day_of_month")
		reg(&gDateTimeUnref, glib, "g_date_time_unref")
		reg(&gtkColorButtonNew, gtk, "gtk_color_button_new")
		reg(&gtkColorChooserGetRGBA, gtk, "gtk_color_chooser_get_rgba")
		reg(&gtkColorChooserSetRGBA, gtk, "gtk_color_chooser_set_rgba")
	})
	return loadErr
}

func dlopenAny(names ...string) (uintptr, error) {
	var last error
	for _, n := range names {
		if h, err := purego.Dlopen(n, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
			return h, nil
		} else {
			last = err
		}
	}
	return 0, fmt.Errorf("gtk4: dlopen %v: %w", names, last)
}

// Init loads GTK4 and initialises it, reporting whether a display could be
// opened. It is safe to call more than once. An error means the libraries could
// not be loaded at all (not merely that there is no display).
func Init() (ok bool, err error) {
	if err := load(); err != nil {
		return false, err
	}
	return gtkInitCheck(), nil
}

// WindowNew creates a top-level window.
func WindowNew() Widget                { return Widget(gtkWindowNew()) }
func (w Widget) SetTitle(title string) { gtkWindowSetTitle(uintptr(w), title) }
func (w Widget) SetDefaultSize(width, height int) {
	gtkWindowSetDefault(uintptr(w), int32(width), int32(height))
}
func (w Widget) SetChild(child Widget) { gtkWindowSetChild(uintptr(w), uintptr(child)) }
func (w Widget) Present()              { gtkWindowPresent(uintptr(w)) }

// FixedNew creates a GtkFixed — the container a host overlays native controls in
// at absolute positions, over the pixel drawing area.
func FixedNew() Widget                           { return Widget(gtkFixedNew()) }
func (f Widget) Put(child Widget, x, y float64)  { gtkFixedPut(uintptr(f), uintptr(child), x, y) }
func (f Widget) Move(child Widget, x, y float64) { gtkFixedMove(uintptr(f), uintptr(child), x, y) }

// ButtonNewWithLabel creates a push button.
func ButtonNewWithLabel(label string) Widget { return Widget(gtkButtonNewWithLabel(label)) }
func (w Widget) SetLabel(label string)       { gtkButtonSetLabel(uintptr(w), label) }

// CheckButtonNewWithLabel creates a labelled check button. GTK check buttons that
// share a group act as radios; a host groups them for a NativeRadio.
func CheckButtonNewWithLabel(label string) Widget { return Widget(gtkCheckButtonNewWith(label)) }

// Active and SetActive read and write the pressed state of a check button OR a
// toggle button — GtkCheckButton and GtkToggleButton are distinct types in GTK4
// with separate accessors, so the method dispatches on the widget's runtime type
// (GTK_IS_TOGGLE_BUTTON). This lets both the check/radio controls and the
// segmented toggles share one Active/SetActive pair.
func (w Widget) Active() bool {
	if w.isToggleButton() {
		return gtkToggleButtonGetActive(uintptr(w))
	}
	return gtkCheckButtonActive(uintptr(w))
}
func (w Widget) SetActive(on bool) {
	if w.isToggleButton() {
		gtkToggleButtonSetActive(uintptr(w), on)
		return
	}
	gtkCheckButtonSetActive(uintptr(w), on)
}

// isToggleButton reports whether w is a GtkToggleButton (as opposed to a
// GtkCheckButton), via g_type_check_instance_is_a against the toggle-button GType.
func (w Widget) isToggleButton() bool {
	return gTypeCheckInstanceIsA(uintptr(w), gtkToggleButtonGetType())
}

// EntryNew creates a single-line text entry. Call SetVisibility(false) for a
// secure (password) entry.
func EntryNew() Widget                  { return Widget(gtkEntryNew()) }
func (w Widget) SetVisibility(vis bool) { gtkEntrySetVisibility(uintptr(w), vis) }

// LabelNew creates a static text label.
func LabelNew(text string) Widget { return Widget(gtkLabelNew(text)) }

// SliderNew creates a horizontal GtkScale over [min,max] with the given step. Read
// and write its value with Value/SetValue; connect "value-changed" for edits.
func SliderNew(min, max, step float64) Widget {
	return Widget(gtkScaleNewWithRange(0, min, max, step)) // 0 = GTK_ORIENTATION_HORIZONTAL
}

// Value and SetValue read and write a slider's (GtkRange's) current value.
func (w Widget) Value() float64     { return gtkRangeGetValue(uintptr(w)) }
func (w Widget) SetValue(v float64) { gtkRangeSetValue(uintptr(w), v) }

// PopUpNew creates a GtkDropDown listing items. The items are copied into a
// GtkStringList (one append per string, so no C string-array marshalling is
// needed). Read and write the selection index with Selected/SetSelected; connect
// "notify::selected" for changes.
func PopUpNew(items []string) Widget {
	list := gtkStringListNew(0) // NULL → empty GtkStringList
	for _, s := range items {
		gtkStringListAppend(list, s)
	}
	return Widget(gtkDropDownNew(list, 0)) // no expression (the model holds strings)
}

// Selected returns the selected index, or -1 when nothing is selected
// (GTK_INVALID_LIST_POSITION). SetSelected selects by index.
func (w Widget) Selected() int {
	if s := gtkDropDownGetSel(uintptr(w)); s != 0xffffffff {
		return int(s)
	}
	return -1
}
func (w Widget) SetSelected(i int) { gtkDropDownSetSel(uintptr(w), uint32(i)) }

// ListBoxNew creates a GtkListBox — a single-column selectable list. Append rows
// with ListBoxAppendText, read/write the selection with SelectedRow/SelectRow, and
// connect "row-selected" for changes.
func ListBoxNew() Widget { return Widget(gtkListBoxNew()) }

// ListBoxAppendText appends a row holding a left-aligned label with text.
func (w Widget) ListBoxAppendText(text string) {
	gtkListBoxAppend(uintptr(w), gtkLabelNew(text))
}

// SelectedRow returns the selected row's index, or -1 when nothing is selected.
func (w Widget) SelectedRow() int {
	row := gtkListBoxGetSelectedRow(uintptr(w))
	if row == 0 {
		return -1
	}
	return int(gtkListBoxRowGetIndex(row))
}

// SelectRow selects the row at index i (a no-op if i is out of range).
func (w Widget) SelectRow(i int) {
	if row := gtkListBoxGetRowAtIndex(uintptr(w), int32(i)); row != 0 {
		gtkListBoxSelectRow(uintptr(w), row)
	}
}

// Text and SetText read and write an editable's text (entry, label via editable
// where applicable), through the GtkEditable interface.
func (w Widget) Text() string     { return gtkEditableGetText(uintptr(w)) }
func (w Widget) SetText(s string) { gtkEditableSetText(uintptr(w), s) }

// SetSizeRequest fixes a widget's size (a host sizes controls to the region it
// laid out).
func (w Widget) SetSizeRequest(width, height int) {
	gtkWidgetSetSizeReq(uintptr(w), int32(width), int32(height))
}

// SetVisible shows or hides a widget in place.
func (w Widget) SetVisible(vis bool) { gtkWidgetSetVisible(uintptr(w), vis) }

// Unparent removes a widget from its parent (a host reconciling controls away).
func (w Widget) Unparent() { gtkWidgetUnparent(uintptr(w)) }

// ProgressNew creates a GtkProgressBar. It is read-only: set its position with
// SetFraction (0..1); there is no change signal (a host drives it).
func ProgressNew() Widget { return Widget(gtkProgressBarNew()) }

// SetFraction sets a progress bar's fill, clamped by GTK to [0,1].
func (w Widget) SetFraction(f float64) { gtkProgressBarSetFraction(uintptr(w), f) }

// SpinnerNew creates a GtkSpinner — an indeterminate activity indicator. Start and
// Stop animate it; it has no value.
func SpinnerNew() Widget { return Widget(gtkSpinnerNew()) }
func (w Widget) Start()  { gtkSpinnerStart(uintptr(w)) }
func (w Widget) Stop()   { gtkSpinnerStop(uintptr(w)) }

// StepperNew creates a GtkSpinButton over [min,max] with the given step. Read and
// write its value with SpinValue/SetSpinValue; connect "value-changed" for edits.
func StepperNew(min, max, step float64) Widget {
	return Widget(gtkSpinButtonNewWithRange(min, max, step))
}
func (w Widget) SpinValue() float64     { return gtkSpinButtonGetValue(uintptr(w)) }
func (w Widget) SetSpinValue(v float64) { gtkSpinButtonSetValue(uintptr(w), v) }

// SearchNew creates a GtkSearchEntry. It implements GtkEditable, so Text/SetText
// work on it directly; connect "search-changed" for edits.
func SearchNew() Widget { return Widget(gtkSearchEntryNew()) }

// ComboNew creates an editable GtkComboBoxText prefilled with items (one append
// per string). Read and write the shown text with ComboText/SetComboText; connect
// "changed" for edits.
func ComboNew(items []string) Widget {
	c := gtkComboBoxTextNewEntry()
	for _, s := range items {
		gtkComboBoxTextAppend(c, s)
	}
	return Widget(c)
}

// ComboText returns the combo's active text — for an editable combo, the text in
// its entry. gtk_combo_box_text_get_active_text returns a freshly g_malloc'd
// string that purego copies into Go; the C copy is not freed, a small acceptable
// per-call leak (no free plumbing, as agreed).
func (w Widget) ComboText() string { return gtkComboBoxTextActiveText(uintptr(w)) }

// SetComboText sets the editable combo's text by writing into its child entry
// (gtk_combo_box_get_child → gtk_editable_set_text).
func (w Widget) SetComboText(s string) { gtkEditableSetText(gtkComboBoxGetChild(uintptr(w)), s) }

// BoxNew creates a GtkBox with zero spacing, horizontal (0) or vertical (1). It is
// the container a host composes a segmented control from, appending linked toggle
// buttons; visual linking (the .linked style class) is the host's job.
func BoxNew(horizontal bool) Widget {
	o := int32(1) // GTK_ORIENTATION_VERTICAL
	if horizontal {
		o = 0 // GTK_ORIENTATION_HORIZONTAL
	}
	return Widget(gtkBoxNew(o, 0))
}

// Append adds child to the end of a GtkBox.
func (w Widget) Append(child Widget) { gtkBoxAppend(uintptr(w), uintptr(child)) }

// ToggleButtonNewWithLabel creates a labelled GtkToggleButton. Group several with
// SetGroup for a mutually-exclusive (segmented) set; read and write pressed state
// with the shared Active/SetActive, and connect "toggled" for changes.
func ToggleButtonNewWithLabel(label string) Widget {
	return Widget(gtkToggleButtonNewLabel(label))
}

// SetGroup joins this toggle button to group's exclusive set. Pass the zero Widget
// to start a fresh group (become the group's first member).
func (w Widget) SetGroup(group Widget) { gtkToggleButtonSetGroup(uintptr(w), uintptr(group)) }

// TextViewNew creates a multi-line GtkTextView. Read and write its whole contents
// with TextViewText/SetTextViewText. The change signal lives on the underlying
// GtkTextBuffer, not the view: connect it with ConnectBufferChanged, or reach the
// buffer with Buffer and connect "changed" yourself.
func TextViewNew() Widget { return Widget(gtkTextViewNew()) }

// Buffer returns the view's GtkTextBuffer (as a Widget handle — it is a GObject a
// host connects signals on, not a displayable widget).
func (w Widget) Buffer() Widget { return Widget(gtkTextViewGetBuffer(uintptr(w))) }

// ConnectBufferChanged connects fn to the view's buffer "changed" signal — fired
// on every edit. It resolves the buffer for the host.
func (w Widget) ConnectBufferChanged(fn func()) uint64 {
	return Widget(gtkTextViewGetBuffer(uintptr(w))).Connect("changed", fn)
}

// SetTextViewText replaces the whole buffer contents.
func (w Widget) SetTextViewText(s string) {
	gtkTextBufferSetText(gtkTextViewGetBuffer(uintptr(w)), s, -1)
}

// TextViewText returns the whole buffer contents. A GtkTextIter is an opaque
// fixed-size struct filled in place; a 128-byte buffer is comfortably larger than
// its ~80 bytes. gtk_text_buffer_get_text returns a freshly g_malloc'd string that
// purego copies into Go; the C copy is not freed, a small acceptable per-call leak.
func (w Widget) TextViewText() string {
	buf := gtkTextViewGetBuffer(uintptr(w))
	var start, end [128]byte
	gtkTextBufferGetStartIter(buf, uintptr(unsafe.Pointer(&start[0])))
	gtkTextBufferGetEndIter(buf, uintptr(unsafe.Pointer(&end[0])))
	return gtkTextBufferGetText(buf, uintptr(unsafe.Pointer(&start[0])), uintptr(unsafe.Pointer(&end[0])), false)
}

// LinkNew creates a GtkLinkButton showing label with an empty URI, so GTK opens
// nothing on its own. Wire OnActivateLink for a plain activation.
func LinkNew(label string) Widget { return Widget(gtkLinkButtonNewLabel("", label)) }

// OnActivateLink connects fn to "activate-link", whose handler returns gboolean.
// The bound callback returns TRUE, which suppresses GTK's own URI-open so the link
// behaves as a plain clickable activation.
func (w Widget) OnActivateLink(fn func()) uint64 {
	cb := purego.NewCallback(func(_ uintptr, _ uintptr) int32 { fn(); return 1 })
	retainCallback(cb)
	return gSignalConnectData(uintptr(w), "activate-link", cb, 0, 0, 0)
}

// DateNew creates a GtkCalendar. Read and write the selected day as "YYYY-MM-DD"
// with DateISO/SetDateISO; connect "day-selected" for changes.
func DateNew() Widget { return Widget(gtkCalendarNew()) }

// DateISO returns the selected day as "YYYY-MM-DD". gtk_calendar_get_date hands
// back a GDateTime we own and unref.
func (w Widget) DateISO() string {
	dt := gtkCalendarGetDate(uintptr(w))
	if dt == 0 {
		return ""
	}
	y, m, d := gDateTimeGetYear(dt), gDateTimeGetMonth(dt), gDateTimeGetDayOfMonth(dt)
	gDateTimeUnref(dt)
	return fmt.Sprintf("%04d-%02d-%02d", y, m, d)
}

// SetDateISO selects the day named by an "YYYY-MM-DD" string. A malformed string
// or an impossible date is ignored.
func (w Widget) SetDateISO(s string) {
	var y, m, d int
	if n, err := fmt.Sscanf(s, "%d-%d-%d", &y, &m, &d); err != nil || n != 3 {
		return
	}
	dt := gDateTimeNewLocal(int32(y), int32(m), int32(d), 0, 0, 0)
	if dt == 0 {
		return
	}
	gtkCalendarSelectDay(uintptr(w), dt)
	gDateTimeUnref(dt)
}

// ColorNew creates a GtkColorButton. Read and write the colour as "#RRGGBB" with
// ColorHex/SetColorHex; connect "color-set" for changes.
func ColorNew() Widget { return Widget(gtkColorButtonNew()) }

// ColorHex returns the chosen colour as "#RRGGBB". GdkRGBA is four C floats
// (red,green,blue,alpha), 16 bytes; alpha is dropped.
func (w Widget) ColorHex() string {
	var rgba [16]byte
	gtkColorChooserGetRGBA(uintptr(w), uintptr(unsafe.Pointer(&rgba[0])))
	r := *(*float32)(unsafe.Pointer(&rgba[0]))
	g := *(*float32)(unsafe.Pointer(&rgba[4]))
	b := *(*float32)(unsafe.Pointer(&rgba[8]))
	return fmt.Sprintf("#%02X%02X%02X", unitToByte(r), unitToByte(g), unitToByte(b))
}

// SetColorHex sets the colour from an "#RRGGBB" string (alpha forced opaque). A
// malformed string is ignored.
func (w Widget) SetColorHex(s string) {
	var r, g, b int
	if n, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); err != nil || n != 3 {
		return
	}
	var rgba [16]byte
	*(*float32)(unsafe.Pointer(&rgba[0])) = float32(r) / 255
	*(*float32)(unsafe.Pointer(&rgba[4])) = float32(g) / 255
	*(*float32)(unsafe.Pointer(&rgba[8])) = float32(b) / 255
	*(*float32)(unsafe.Pointer(&rgba[12])) = 1 // alpha
	gtkColorChooserSetRGBA(uintptr(w), uintptr(unsafe.Pointer(&rgba[0])))
}

// unitToByte maps a [0,1] colour component to a rounded 0..255 byte.
func unitToByte(v float32) int {
	n := int(float64(v)*255 + 0.5)
	if n < 0 {
		return 0
	}
	if n > 255 {
		return 255
	}
	return n
}

// Connect wires a GObject signal (e.g. "clicked", "changed") to a Go func. The
// callback keeps the widget's value reachable through its own accessors, so the
// zero-argument closure is enough for the control signals a host cares about. The
// returned handler id is GObject's; a host rarely needs it.
//
// The callback is retained for the process life (like the go-macos target
// classes): GTK holds a C pointer to it, and letting Go collect it would leave a
// dangling call.
func (w Widget) Connect(signal string, fn func()) uint64 {
	cb := purego.NewCallback(func(_ uintptr, _ uintptr) { fn() })
	retainCallback(cb)
	return gSignalConnectData(uintptr(w), signal, cb, 0, 0, 0)
}

// MainLoopNew creates a GLib main loop.
func MainLoopNew() MainLoop { return MainLoop(gMainLoopNew(0, false)) }
func (l MainLoop) Run()     { gMainLoopRun(uintptr(l)) }
func (l MainLoop) Quit()    { gMainLoopQuit(uintptr(l)) }

// IdleAdd schedules fn to run once on the main loop and be removed. A host uses
// it to marshal work onto the GTK thread. fn runs on the main loop thread.
func IdleAdd(fn func()) {
	cb := purego.NewCallback(func(_ uintptr) int32 { fn(); return 0 })
	retainCallback(cb)
	gIdleAdd(cb, 0)
}

// AddTickCallback registers fn to run once per frame, driven by this widget's
// GdkFrameClock, for as long as fn returns true. It is the GTK-native animation
// tick: aligned to the display's refresh, and quiescent while the widget is
// unmapped (the frame clock does not run then), so an idle hidden window costs
// nothing. A host that renders its own pixels into a [Picture] uses it to present
// a fresh frame each vsync — unlike a one-shot [IdleAdd], which the frame clock
// would fire before the window is even mapped. fn runs on the main-loop thread;
// returning false removes the callback. The callback is retained for the process
// life, like [Widget.Connect]'s.
func (w Widget) AddTickCallback(fn func() bool) uint64 {
	cb := purego.NewCallback(func(_ uintptr, _ uintptr, _ uintptr) int32 {
		if fn() {
			return 1 // G_SOURCE_CONTINUE — keep ticking
		}
		return 0 // G_SOURCE_REMOVE
	})
	retainCallback(cb)
	return uint64(gtkWidgetAddTick(uintptr(w), cb, 0, 0))
}

// retainCallback keeps every callback alive for the process: GTK stores the C
// function pointer and may call it at any later time.
var (
	cbMu   sync.Mutex
	cbKeep []uintptr
)

func retainCallback(cb uintptr) {
	cbMu.Lock()
	cbKeep = append(cbKeep, cb)
	cbMu.Unlock()
}
