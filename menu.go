// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package gtk4

import "strings"

// A MenuItem is one entry of a [ContextMenu]. An empty Label is a separator. A
// nil OnPick is an inert entry: shown, greyed out, and never called.
type MenuItem struct {
	Label  string
	OnPick func()
}

// A MenuEntry is one chosen-able line of a menu model: the position of its item
// in the caller's list (what an activation reports), the label GTK shows, and
// whether it can be chosen.
type MenuEntry struct {
	Index   int
	Label   string
	Enabled bool
}

// MenuSections lays a flat item list out the way a GMenu wants it: runs of
// entries, one run per section, a separator being the boundary between two runs.
// A leading, trailing or doubled separator makes no empty run, because GTK draws
// a rule for each section and an empty one is a rule beside nothing.
//
// The labels come back ready for GTK, which reads a leading-underscore pair as a
// mnemonic: "Save _as" would underline the "a" and drop the underscore, so every
// underscore is doubled and the person sees the word the application wrote.
func MenuSections(items []MenuItem) [][]MenuEntry {
	var out [][]MenuEntry
	var run []MenuEntry
	flush := func() {
		if len(run) > 0 {
			out = append(out, run)
			run = nil
		}
	}
	for i, it := range items {
		if it.Label == "" {
			flush()
			continue
		}
		run = append(run, MenuEntry{
			Index:   i,
			Label:   strings.ReplaceAll(it.Label, "_", "__"),
			Enabled: it.OnPick != nil,
		})
	}
	flush()
	return out
}
