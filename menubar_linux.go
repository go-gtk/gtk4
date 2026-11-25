// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

var (
	barOnce sync.Once
	barErr  error

	gMenuAppendSubmenu      func(uintptr, string, uintptr)
	gSimpleActionNewState   func(string, uintptr, uintptr) uintptr
	gSimpleActionSetState   func(uintptr, uintptr)
	gVariantNewBoolean      func(bool) uintptr
	gtkPopoverMenuBarNew    func(uintptr) uintptr
	gtkShortcutCtlNew       func() uintptr
	gtkShortcutCtlScope     func(uintptr, int32)
	gtkShortcutCtlAdd       func(uintptr, uintptr)
	gtkShortcutTriggerParse func(string) uintptr
	gtkNamedActionNew       func(string) uintptr
	gtkShortcutNew          func(uintptr, uintptr) uintptr
	gtkWidgetRemoveCtl      func(uintptr, uintptr)
	gtkBoxPrepend           func(uintptr, uintptr)
	gtkSetHExpand           func(uintptr, bool)
	gtkSetVExpand           func(uintptr, bool)
)

func loadBar() error {
	barOnce.Do(func() {
		if barErr = loadMenu(); barErr != nil {
			return
		}
		gio, err := dlopenAny("libgio-2.0.so.0", "libgio-2.0.so")
		if err != nil {
			barErr = err
			return
		}
		glib, err := dlopenAny("libglib-2.0.so.0", "libglib-2.0.so")
		if err != nil {
			barErr = err
			return
		}
		gtk, err := dlopenAny("libgtk-4.so.1", "libgtk-4.so")
		if err != nil {
			barErr = err
			return
		}
		reg := func(p any, h uintptr, name string) { purego.RegisterLibFunc(p, h, name) }
		reg(&gMenuAppendSubmenu, gio, "g_menu_append_submenu")
		reg(&gSimpleActionNewState, gio, "g_simple_action_new_stateful")
		reg(&gSimpleActionSetState, gio, "g_simple_action_set_state")
		reg(&gVariantNewBoolean, glib, "g_variant_new_boolean")
		reg(&gtkPopoverMenuBarNew, gtk, "gtk_popover_menu_bar_new_from_model")
		reg(&gtkShortcutCtlNew, gtk, "gtk_shortcut_controller_new")
		reg(&gtkShortcutCtlScope, gtk, "gtk_shortcut_controller_set_scope")
		reg(&gtkShortcutCtlAdd, gtk, "gtk_shortcut_controller_add_shortcut")
		reg(&gtkShortcutTriggerParse, gtk, "gtk_shortcut_trigger_parse_string")
		reg(&gtkNamedActionNew, gtk, "gtk_named_action_new")
		reg(&gtkShortcutNew, gtk, "gtk_shortcut_new")
		reg(&gtkWidgetRemoveCtl, gtk, "gtk_widget_remove_controller")
		reg(&gtkBoxPrepend, gtk, "gtk_box_prepend")
		reg(&gtkSetHExpand, gtk, "gtk_widget_set_hexpand")
		reg(&gtkSetVExpand, gtk, "gtk_widget_set_vexpand")
	})
	return barErr
}

// A MenuBar is a GtkPopoverMenuBar over a GMenu, the application menu bar of a
// plain GtkWindow (no GtkApplication needed). Its actions live in the "bar"
// group inserted on the window, and its accelerators are a GtkShortcutController
// on the window, so a chord works wherever the keyboard focus is.
//
// Build it with [MenuBarNew], put [MenuBar.Widget] where the bar should sit
// (typically prepended to a vertical box) and call [MenuBar.SetItems] on the
// main thread whenever the rows change.
type MenuBar struct {
	id     uint32
	win    Widget
	bar    uintptr
	model  uintptr
	ctl    uintptr // the window's current shortcut controller, 0 before the first SetItems
	leaves []barLeaf
}

var barReg = struct {
	sync.Mutex
	next uint32
	m    map[uint32]*MenuBar
}{m: map[uint32]*MenuBar{}}

var barCbOnce struct {
	sync.Once
	activate, change uintptr
}

func barLookup(id uint32) *MenuBar {
	barReg.Lock()
	defer barReg.Unlock()
	return barReg.m[id]
}

func barCallbacks() (activate, change uintptr) {
	barCbOnce.Do(func() {
		// data is bar id << 20 | leaf index.
		find := func(data uintptr) (*MenuBar, int) {
			m := barLookup(uint32(data >> 20))
			idx := int(data & (1<<20 - 1))
			if m == nil || idx >= len(m.leaves) {
				return nil, 0
			}
			return m, idx
		}
		barCbOnce.activate = purego.NewCallback(func(_, _, data uintptr) {
			if m, i := find(data); m != nil && m.leaves[i].item.OnActivate != nil {
				m.leaves[i].item.OnActivate()
			}
		})
		barCbOnce.change = purego.NewCallback(func(act, _, data uintptr) {
			m, i := find(data)
			if m == nil {
				return
			}
			it := m.leaves[i].item
			if it.OnActivate != nil {
				it.OnActivate()
			}
			// Hold the state at what the host says; it moves it by SetItems.
			gSimpleActionSetState(act, gVariantNewBoolean(it.Checked))
		})
		retainCallback(barCbOnce.activate)
		retainCallback(barCbOnce.change)
	})
	return barCbOnce.activate, barCbOnce.change
}

// MenuBarSupported reports whether this GTK has what a [MenuBar] needs.
func MenuBarSupported() bool { return loadBar() == nil }

// MenuBarNew makes an empty bar for window. It returns nil when the menu
// machinery cannot be loaded.
func MenuBarNew(window Widget) *MenuBar {
	if window == 0 || loadBar() != nil {
		return nil
	}
	barReg.Lock()
	barReg.next++
	m := &MenuBar{win: window, model: gMenuNew(), id: barReg.next}
	barReg.m[m.id] = m
	barReg.Unlock()
	m.bar = gtkPopoverMenuBarNew(m.model)
	return m
}

// Prepend adds child to the start of a GtkBox, which is where a menu bar goes.
func (w Widget) Prepend(child Widget) {
	if loadBar() == nil {
		gtkBoxPrepend(uintptr(w), uintptr(child))
	}
}

// SetExpand makes the widget take the space its parent has left over, across
// (h) and down (v): what keeps a box's main child filling the window under a bar.
func (w Widget) SetExpand(h, v bool) {
	if loadBar() == nil {
		gtkSetHExpand(uintptr(w), h)
		gtkSetVExpand(uintptr(w), v)
	}
}

// Widget is the GtkPopoverMenuBar to place in the window.
func (m *MenuBar) Widget() Widget { return Widget(m.bar) }

// Len is the number of pickable rows (commands and checks, not menus).
func (m *MenuBar) Len() int { return len(m.leaves) }

// SetItems replaces the bar's menus. It does the whole swap, so call it only
// when something changed.
func (m *MenuBar) SetItems(items []BarItem) {
	secs, leaves := planBar(items)
	m.leaves = leaves
	activate, change := barCallbacks()

	group := gSimpleActionGroupNew()
	ctl := gtkShortcutCtlNew()
	gtkShortcutCtlScope(ctl, 2) // GTK_SHORTCUT_SCOPE_GLOBAL: the root handles it
	// Capture phase: a painted area below consumes keys in the bubble phase, and
	// a menu accelerator must win over it.
	gtkCtrlSetPhase(ctl, 1) // GTK_PHASE_CAPTURE
	for i := range m.leaves {
		l := &m.leaves[i]
		l.name = fmt.Sprintf("i%d", i)
		var act uintptr
		if l.item.Check {
			act = gSimpleActionNewState(l.name, 0, gVariantNewBoolean(l.item.Checked))
			gSignalConnectData(act, "change-state", change, uintptr(m.id)<<20|uintptr(i), 0, 0)
		} else {
			act = gSimpleActionNew(l.name, 0)
			gSignalConnectData(act, "activate", activate, uintptr(m.id)<<20|uintptr(i), 0, 0)
		}
		gSimpleActionEnabled(act, !l.item.Disabled && l.item.OnActivate != nil)
		gActionMapAdd(group, act)
		gObjectUnref(act) // the group holds it now
		if l.item.Accel != "" {
			if trig := gtkShortcutTriggerParse(l.item.Accel); trig != 0 {
				gtkShortcutCtlAdd(ctl, gtkShortcutNew(trig, gtkNamedActionNew("bar."+l.name)))
			}
		}
	}

	gMenuRemoveAll(m.model)
	m.fill(m.model, secs)
	gtkInsertActionGroup(uintptr(m.win), "bar", group)
	gObjectUnref(group)
	if m.ctl != 0 {
		gtkWidgetRemoveCtl(uintptr(m.win), m.ctl)
	}
	gtkWidgetAddController(uintptr(m.win), ctl)
	m.ctl = ctl
}

// fill appends secs to menu. The bar's root is a list of menus, so its sections
// are flattened: a root has no rules between its menus.
func (m *MenuBar) fill(menu uintptr, secs []barSection) {
	root := menu == m.model
	for _, sec := range secs {
		target := menu
		var section uintptr
		if !root {
			section = gMenuNew()
			target = section
		}
		for _, n := range sec {
			if n.leaf < 0 {
				sub := gMenuNew()
				m.fill(sub, n.sub)
				gMenuAppendSubmenu(target, n.label, sub)
				gObjectUnref(sub)
				continue
			}
			gMenuAppend(target, n.label, "bar."+m.leaves[n.leaf].name)
		}
		if !root {
			gMenuAppendSection(menu, 0, section)
			gObjectUnref(section)
		}
	}
}

// Activate chooses leaf i (depth-first order) through its action name, as a click
// or its accelerator would, and reports whether it was a row that can be chosen.
func (m *MenuBar) Activate(i int) bool {
	if i < 0 || i >= len(m.leaves) {
		return false
	}
	// GTK reports a disabled action as found, so the refusal is made here.
	if it := m.leaves[i].item; it.Disabled || it.OnActivate == nil {
		return false
	}
	return gtkActivateAction(uintptr(m.win), "bar."+m.leaves[i].name, 0)
}

// Destroy drops the bar's actions and shortcuts from the window. The bar widget
// itself is the caller's to unparent.
func (m *MenuBar) Destroy() {
	gtkInsertActionGroup(uintptr(m.win), "bar", 0)
	if m.ctl != 0 {
		gtkWidgetRemoveCtl(uintptr(m.win), m.ctl)
		m.ctl = 0
	}
	barReg.Lock()
	delete(barReg.m, m.id)
	barReg.Unlock()
}
