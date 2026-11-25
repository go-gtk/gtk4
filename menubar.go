// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package gtk4

import "strings"

// A BarItem is one row of a [MenuBar]. The bar's own items are menus (Sub set);
// inside a menu a row is a command, a check, a submenu (Sub set) or a Separator.
type BarItem struct {
	Label     string
	Separator bool
	Disabled  bool
	// Check makes the row a stateful boolean action, drawn with a tick while
	// Checked. A pick calls OnActivate and then puts the state back to Checked,
	// so the row shows what the host says, not what GTK toggled it to: the host
	// changes Checked and calls [MenuBar.SetItems] again.
	Check   bool
	Checked bool
	// Accel is a GTK accelerator string such as "<Control>s" or "<Shift>F5". It
	// is bound on the bar's window and drawn beside the label by GTK.
	Accel string
	// OnActivate runs on the GTK main thread when the row is chosen or its
	// accelerator pressed. Rows with a nil OnActivate are shown greyed out.
	OnActivate func()
	// Sub, when non-nil, makes the row a menu holding these rows.
	Sub []BarItem
}

// barEscape doubles underscores, which GTK reads as mnemonic prefixes.
func barEscape(s string) string { return strings.ReplaceAll(s, "_", "__") }

// barLeaf is a pickable row in depth-first order.
type barLeaf struct {
	item BarItem
	name string // the action name inside the "bar" group
}

// barSection is one run of rows between separators, ready to become a GMenu
// section.
type barSection []barNode

// barNode is a leaf (by index into the leaf list) or, with leaf == -1, a submenu.
type barNode struct {
	label string
	leaf  int
	sub   []barSection
}

// planBar lays items out for GMenu: sections split at separators (no empty
// ones), leaves numbered depth-first. It is pure, so it is tested without GTK.
func planBar(items []BarItem) (secs []barSection, leaves []barLeaf) {
	var walk func([]BarItem) []barSection
	walk = func(in []BarItem) []barSection {
		var out []barSection
		var run barSection
		flush := func() {
			if len(run) > 0 {
				out = append(out, run)
				run = nil
			}
		}
		for _, it := range in {
			switch {
			case it.Separator:
				flush()
			case it.Sub != nil:
				run = append(run, barNode{label: barEscape(it.Label), leaf: -1, sub: walk(it.Sub)})
			default:
				leaves = append(leaves, barLeaf{item: it})
				run = append(run, barNode{label: barEscape(it.Label), leaf: len(leaves) - 1})
			}
		}
		flush()
		return out
	}
	secs = walk(items)
	return secs, leaves
}
