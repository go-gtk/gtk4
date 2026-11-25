// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	menuOnce sync.Once
	menuErr  error

	gMenuNew              func() uintptr
	gMenuAppend           func(uintptr, string, string)
	gMenuAppendSection    func(uintptr, uintptr, uintptr)
	gMenuRemoveAll        func(uintptr)
	gSimpleActionGroupNew func() uintptr
	gSimpleActionNew      func(string, uintptr) uintptr
	gSimpleActionEnabled  func(uintptr, bool)
	gActionMapAdd         func(uintptr, uintptr)
	gtkActivateAction     func(uintptr, string, uintptr) bool
	gtkInsertActionGroup  func(uintptr, string, uintptr)
	gtkPopoverMenuNew     func(uintptr) uintptr
	gtkPopoverSetParent   func(uintptr, uintptr)
	gtkPopoverPointingTo  func(uintptr, uintptr)
	gtkPopoverHasArrow    func(uintptr, bool)
	gtkPopoverPopup       func(uintptr)
	gtkPopoverPopdown     func(uintptr)
	gtkWidgetGetMapped    func(uintptr) bool
	gtkGestureSetState    func(uintptr, int32) bool
	gtkCtrlSetPhase       func(uintptr, int32)
	gtkWindowClose        func(uintptr)
	gtkWidgetSetTooltip   func(uintptr, string)
	gtkButtonSetChild     func(uintptr, uintptr)
	gtkPictureSetCanShrk  func(uintptr, bool)
	gtkPictureSetAlt      func(uintptr, string)
)

func loadMenu() error {
	menuOnce.Do(func() {
		if menuErr = load(); menuErr != nil {
			return
		}
		if menuErr = loadPixels(); menuErr != nil {
			return
		}
		gio, err := dlopenAny("libgio-2.0.so.0", "libgio-2.0.so")
		if err != nil {
			menuErr = err
			return
		}
		gtk, err := dlopenAny("libgtk-4.so.1", "libgtk-4.so")
		if err != nil {
			menuErr = err
			return
		}
		reg := func(p any, h uintptr, name string) { purego.RegisterLibFunc(p, h, name) }
		reg(&gMenuNew, gio, "g_menu_new")
		reg(&gMenuAppend, gio, "g_menu_append")
		reg(&gMenuAppendSection, gio, "g_menu_append_section")
		reg(&gMenuRemoveAll, gio, "g_menu_remove_all")
		reg(&gSimpleActionGroupNew, gio, "g_simple_action_group_new")
		reg(&gSimpleActionNew, gio, "g_simple_action_new")
		reg(&gSimpleActionEnabled, gio, "g_simple_action_set_enabled")
		reg(&gActionMapAdd, gio, "g_action_map_add_action")
		reg(&gtkActivateAction, gtk, "gtk_widget_activate_action_variant")
		reg(&gtkInsertActionGroup, gtk, "gtk_widget_insert_action_group")
		reg(&gtkPopoverMenuNew, gtk, "gtk_popover_menu_new_from_model")
		reg(&gtkPopoverSetParent, gtk, "gtk_widget_set_parent")
		reg(&gtkPopoverPointingTo, gtk, "gtk_popover_set_pointing_to")
		reg(&gtkPopoverHasArrow, gtk, "gtk_popover_set_has_arrow")
		reg(&gtkPopoverPopup, gtk, "gtk_popover_popup")
		reg(&gtkPopoverPopdown, gtk, "gtk_popover_popdown")
		reg(&gtkWidgetGetMapped, gtk, "gtk_widget_get_mapped")
		reg(&gtkGestureSetState, gtk, "gtk_gesture_set_state")
		reg(&gtkCtrlSetPhase, gtk, "gtk_event_controller_set_propagation_phase")
		reg(&gtkWindowClose, gtk, "gtk_window_close")
		reg(&gtkWidgetSetTooltip, gtk, "gtk_widget_set_tooltip_text")
		reg(&gtkButtonSetChild, gtk, "gtk_button_set_child")
		reg(&gtkPictureSetCanShrk, gtk, "gtk_picture_set_can_shrink")
		reg(&gtkPictureSetAlt, gtk, "gtk_picture_set_alternative_text")
	})
	return menuErr
}

// A ContextMenu is a GtkPopoverMenu over a GMenu, parented to a widget and
// opened by a secondary click on it (or by [ContextMenu.Popup]).
//
// It is built so a host that reconciles its controls every frame can keep one
// per control: [ContextMenu.SetItems] swaps the entries in place, and a menu
// that is open when it is called stays open under the pointer.
type ContextMenu struct {
	id      uint32
	parent  Widget
	model   uintptr // the root GMenu
	popover uintptr
	items   []MenuItem
	secs    [][]MenuEntry
}

// ContextMenuNew attaches an (initially empty) context menu to parent: a
// secondary press on parent or anything inside it opens the menu at the pointer
// while it has entries, and is left to the widget's own handling while it has
// none. It returns nil when the menu machinery cannot be loaded.
//
// Destroy it before the widget is unparented: GTK warns about, and leaks, a
// popover still attached to a widget that is finalized.
func ContextMenuNew(parent Widget) *ContextMenu {
	if parent == 0 || loadMenu() != nil {
		return nil
	}
	menuReg.Lock()
	menuReg.next++
	m := &ContextMenu{parent: parent, model: gMenuNew(), id: menuReg.next}
	menuReg.m[m.id] = m
	menuReg.Unlock()
	m.popover = gtkPopoverMenuNew(m.model)
	gtkPopoverSetParent(m.popover, uintptr(parent))
	gtkPopoverHasArrow(m.popover, false)

	g := gtkGestureClickNew()
	gtkGestureSingleSetBtn(g, 3) // the secondary button only
	// Capture, so the menu wins over a child that has a menu of its own (an
	// entry's cut/copy/paste) when the application gave this control one.
	gtkCtrlSetPhase(g, 1) // GTK_PHASE_CAPTURE
	gtkWidgetAddController(uintptr(parent), g)
	gSignalConnectData(g, "pressed", menuCallbacks().press, uintptr(m.id), 0, 0)
	return m
}

// menuReg maps the small integer handed to GTK as a signal's user data back to
// its menu. GTK is given a number, never a Go pointer, and the callbacks below
// are made once for the process: purego's callback table is finite, and a host
// that rebuilds a menu per control must not spend an entry each time.
var menuReg = struct {
	sync.Mutex
	next uint32
	m    map[uint32]*ContextMenu
}{m: map[uint32]*ContextMenu{}}

type menuCbs struct{ press, activate uintptr }

var menuCbOnce struct {
	sync.Once
	cbs menuCbs
}

func menuLookup(id uint32) *ContextMenu {
	menuReg.Lock()
	defer menuReg.Unlock()
	return menuReg.m[id]
}

func menuCallbacks() menuCbs {
	menuCbOnce.Do(func() {
		press := purego.NewCallback(func(g uintptr, _ int32, x, y float64, data uintptr) {
			if m := menuLookup(uint32(data)); m != nil && m.press(x, y) {
				gtkGestureSetState(g, 1) // GTK_EVENT_SEQUENCE_CLAIMED: nothing below opens its own
			}
		})
		activate := purego.NewCallback(func(_, _, data uintptr) {
			// data is menu id << 20 | item index. The live list is read at the
			// time of the pick, not the one the entry was built for: the host
			// swaps its handlers every frame.
			m := menuLookup(uint32(data >> 20))
			idx := int(data & (1<<20 - 1))
			if m != nil && idx < len(m.items) && m.items[idx].OnPick != nil {
				m.items[idx].OnPick()
			}
		})
		retainCallback(press)
		retainCallback(activate)
		menuCbOnce.cbs = menuCbs{press: press, activate: activate}
	})
	return menuCbOnce.cbs
}

// press opens the menu at (x, y) when there is one to open, and reports whether
// it did.
func (m *ContextMenu) press(x, y float64) bool {
	if len(m.secs) == 0 {
		return false
	}
	m.Popup(x, y)
	return true
}

// SetItems replaces the menu's entries. It does the whole swap and nothing
// else, so call it only when the words changed -- see the host's change test.
func (m *ContextMenu) SetItems(items []MenuItem) {
	m.items = append(m.items[:0:0], items...)
	m.secs = MenuSections(m.items)

	gMenuRemoveAll(m.model)
	group := gSimpleActionGroupNew()
	for _, sec := range m.secs {
		section := gMenuNew()
		for _, e := range sec {
			name := fmt.Sprintf("i%d", e.Index)
			act := gSimpleActionNew(name, 0)
			gSimpleActionEnabled(act, e.Enabled)
			gSignalConnectData(act, "activate", menuCallbacks().activate,
				uintptr(m.id)<<20|uintptr(e.Index), 0, 0)
			gActionMapAdd(group, act)
			gObjectUnref(act) // the group holds it now
			gMenuAppend(section, e.Label, "ctx."+name)
		}
		gMenuAppendSection(m.model, 0, section)
		gObjectUnref(section)
	}
	// Replaces the previous group under the same prefix; GTK drops the old one.
	gtkInsertActionGroup(uintptr(m.parent), "ctx", group)
	gObjectUnref(group)
}

// Popup opens the menu with its corner at the widget-local point (x, y).
func (m *ContextMenu) Popup(x, y float64) {
	r := [4]int32{int32(x), int32(y), 1, 1} // GdkRectangle
	gtkPopoverPointingTo(m.popover, uintptr(unsafe.Pointer(&r[0])))
	gtkPopoverPopup(m.popover)
}

// Popdown closes the menu if it is open.
func (m *ContextMenu) Popdown() { gtkPopoverPopdown(m.popover) }

// IsOpen reports whether the menu is on screen.
func (m *ContextMenu) IsOpen() bool { return gtkWidgetGetMapped(m.popover) }

// Len is the number of entries the menu offers, separators not counted.
func (m *ContextMenu) Len() int {
	n := 0
	for _, s := range m.secs {
		n += len(s)
	}
	return n
}

// Activate chooses the item at position i of the list given to SetItems, as a
// click on it would, and reports whether it was an entry that could be chosen.
func (m *ContextMenu) Activate(i int) bool {
	if i < 0 || i >= len(m.items) || m.items[i].Label == "" || m.items[i].OnPick == nil {
		return false
	}
	// Through the widget, by its action name, so what is activated is what a
	// click on the entry would reach.
	return gtkActivateAction(uintptr(m.parent), fmt.Sprintf("ctx.i%d", i), 0)
}

// Destroy closes the menu and detaches it from its widget. The ContextMenu must
// not be used afterwards.
func (m *ContextMenu) Destroy() {
	gtkPopoverPopdown(m.popover)
	Widget(m.popover).Unparent()
	m.popover = 0
	gtkInsertActionGroup(uintptr(m.parent), "ctx", 0)
	menuReg.Lock()
	delete(menuReg.m, m.id)
	menuReg.Unlock()
}

// OnCloseRequest runs fn when the window's own close control is used, and lets
// the window close only if fn returns false. Returning true handles the request:
// the window stays, and is the caller's to hide.
func (w Widget) OnCloseRequest(fn func() (stop bool)) {
	if load() != nil {
		return
	}
	cb := purego.NewCallback(func(_ uintptr, _ uintptr) int32 {
		if fn() {
			return 1 // GDK_EVENT_STOP: the default handler, which destroys, is skipped
		}
		return 0
	})
	retainCallback(cb)
	gSignalConnectData(uintptr(w), "close-request", cb, 0, 0, 0)
}

// Close asks a window to close, as its close control does: the "close-request"
// signal runs, and the window is destroyed unless a handler stopped it.
func (w Widget) Close() {
	if loadMenu() == nil {
		gtkWindowClose(uintptr(w))
	}
}

// Mapped reports whether the widget is on screen.
func (w Widget) Mapped() bool { return loadMenu() == nil && gtkWidgetGetMapped(uintptr(w)) }

// SetTooltip sets the text shown on hover, which assistive technology also reads
// for a widget with no text of its own.
func (w Widget) SetTooltip(text string) {
	if loadMenu() == nil {
		gtkWidgetSetTooltip(uintptr(w), text)
	}
}

// SetButtonChild replaces a button's content with child (the null widget gives
// the button back its label). A button given a child no longer draws its label.
func (w Widget) SetButtonChild(child Widget) {
	if loadMenu() == nil {
		gtkButtonSetChild(uintptr(w), uintptr(child))
	}
}

// SetCanShrink lets the picture be drawn smaller than the image, which is what
// makes a size request on it take effect.
func (p Picture) SetCanShrink(v bool) {
	if loadMenu() == nil {
		gtkPictureSetCanShrk(uintptr(p), v)
	}
}

// SetAlternativeText is the text a screen reader speaks for the picture.
func (p Picture) SetAlternativeText(text string) {
	if loadMenu() == nil {
		gtkPictureSetAlt(uintptr(p), text)
	}
}
