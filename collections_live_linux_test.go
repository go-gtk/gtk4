// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import (
	"reflect"
	"strconv"
	"testing"
)

func collInit(t *testing.T) {
	t.Helper()
	ok, err := Init()
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if !ok {
		t.Skip("no display; run under Xvfb")
	}
}

func manyRows(n int) [][]string {
	rows := make([][]string, n)
	for i := range rows {
		rows[i] = []string{"row " + strconv.Itoa(i), strconv.Itoa(i * 2), "x"}
	}
	return rows
}

// TestLiveColumnView proves a ColumnView holds 10 000 rows as a GtkColumnView,
// round-trips its selection both ways without echoing the host's own change, and
// replaces the rows in one step.
func TestLiveColumnView(t *testing.T) {
	collInit(t)
	v := ColumnViewNew([]string{"Name", "Double", "Tag"})
	if v == nil {
		t.Fatal("ColumnViewNew returned nil")
	}
	defer v.Destroy()
	if got := v.Class(); got != "GtkColumnView" {
		t.Errorf("class = %q, want GtkColumnView", got)
	}
	var reported []int
	v.OnSelect(func(i int) { reported = append(reported, i) })
	v.SetRows(manyRows(10000))
	if got := cListModelNItems(v.list); got != 10000 || v.Len() != 10000 {
		t.Errorf("model holds %d rows, Len %d; want 10000", got, v.Len())
	}
	if v.Selected() != -1 {
		t.Errorf("fresh selection = %d, want -1", v.Selected())
	}
	v.SetSelected(5)
	if v.Selected() != 5 || len(reported) != 0 {
		t.Errorf("SetSelected(5): selected %d, reported %v (a host's own change is not echoed)", v.Selected(), reported)
	}
	// The person picks row 7: reported.
	v.Select(7)
	if !reflect.DeepEqual(reported, []int{7}) || v.Selected() != 7 {
		t.Errorf("after the person selected 7: reported %v, selected %d", reported, v.Selected())
	}
	v.SetSelected(-1)
	if v.Selected() != -1 || len(reported) != 1 {
		t.Errorf("after clearing: selected %d, reported %v", v.Selected(), reported)
	}
	v.SetSelected(10000) // out of range clears, not a wrap
	if v.Selected() != -1 {
		t.Errorf("an out-of-range row selected %d", v.Selected())
	}
	before := v.Rebuilds()
	v.SetRows(manyRows(9000))
	if v.Rebuilds() != before+1 || cListModelNItems(v.list) != 9000 {
		t.Errorf("rebuilds +%d, rows %d; want +1 and 9000", v.Rebuilds()-before, cListModelNItems(v.list))
	}
	v.SetRows(nil)
	if v.Len() != 0 || cListModelNItems(v.list) != 0 {
		t.Error("SetRows(nil) left rows behind")
	}
}

// TestLiveTreeView proves a TreeView starts Expanded nodes open, builds children
// lazily, round-trips the selected path and opens ancestors to show a selection.
func TestLiveTreeView(t *testing.T) {
	collInit(t)
	tv := TreeViewNew()
	if tv == nil {
		t.Fatal("TreeViewNew returned nil")
	}
	defer tv.Destroy()
	if got := tv.Class(); got != "GtkListView" {
		t.Errorf("class = %q, want GtkListView", got)
	}
	var reported [][]int
	tv.OnSelect(func(p []int) { reported = append(reported, p) })
	tv.SetNodes([]TreeNode{
		{Title: "A", Expanded: true, Children: []TreeNode{
			{Title: "A0"},
			{Title: "A1", Children: []TreeNode{{Title: "deep"}}},
		}},
		{Title: "B"},
	})
	// A is expanded, A1 is not: A, A0, A1, B.
	if got := cListModelNItems(tv.tlm); got != 4 {
		t.Errorf("visible rows = %d, want 4 (A, A0, A1, B)", got)
	}
	tv.SetSelectedPath([]int{0, 1})
	if got := tv.SelectedPath(); !reflect.DeepEqual(got, []int{0, 1}) || len(reported) != 0 {
		t.Errorf("SetSelectedPath: selected %v, reported %v", got, reported)
	}
	// The person picks B (position 3).
	tv.SelectPath([]int{1})
	if len(reported) != 1 || !reflect.DeepEqual(reported[0], []int{1}) {
		t.Errorf("after the person selected B, reported = %v", reported)
	}
	// A node under a collapsed parent: the parent opens to show it.
	tv.SetSelectedPath([]int{0, 1, 0})
	if got := cListModelNItems(tv.tlm); got != 5 {
		t.Errorf("after selecting 0.1.0 the visible rows = %d, want 5", got)
	}
	if got := tv.SelectedPath(); !reflect.DeepEqual(got, []int{0, 1, 0}) || len(reported) != 1 {
		t.Errorf("selected %v, reported %v", got, reported)
	}
	// The person picking a node under a collapsed parent is one report, with the
	// final path, though the ancestors open on the way.
	tv.SetNodes([]TreeNode{{Title: "P", Children: []TreeNode{{Title: "Q", Children: []TreeNode{{Title: "R"}}}}}})
	reported = nil
	tv.SelectPath([]int{0, 0, 0})
	if len(reported) != 1 || !reflect.DeepEqual(reported[0], []int{0, 0, 0}) || cListModelNItems(tv.tlm) != 3 {
		t.Errorf("after the person selected 0.0.0: reported %v, rows %d", reported, cListModelNItems(tv.tlm))
	}
	tv.SetSelectedPath([]int{9})
	if tv.SelectedPath() != nil {
		t.Errorf("a path naming nothing selected %v", tv.SelectedPath())
	}
	before := tv.Rebuilds()
	tv.SetNodes([]TreeNode{{Title: "only"}})
	if tv.Rebuilds() != before+1 || cListModelNItems(tv.tlm) != 1 {
		t.Errorf("rebuilds +%d, rows %d; want +1 and 1", tv.Rebuilds()-before, cListModelNItems(tv.tlm))
	}
}

// TestLiveNotebook proves a Notebook shows one page per title, round-trips its
// current page without echoing the host's own change, and removes pages.
func TestLiveNotebook(t *testing.T) {
	collInit(t)
	n := NotebookNew()
	if n == nil {
		t.Fatal("NotebookNew returned nil")
	}
	defer n.Destroy()
	if got := n.Class(); got != "GtkNotebook" {
		t.Errorf("class = %q, want GtkNotebook", got)
	}
	var reported []int
	n.OnSwitch(func(i int) { reported = append(reported, i) })
	n.SetTabs([]string{"One", "Two", "Three"})
	if n.Len() != 3 {
		t.Fatalf("pages = %d, want 3", n.Len())
	}
	n.SetCurrent(1)
	if n.Current() != 1 || len(reported) != 0 {
		t.Errorf("SetCurrent(1): current %d, reported %v", n.Current(), reported)
	}
	n.Switch(2) // the person clicks the third tab
	if !reflect.DeepEqual(reported, []int{2}) || n.Current() != 2 {
		t.Errorf("after the person picked tab 2: reported %v, current %d", reported, n.Current())
	}
	n.SetCurrent(7) // out of range is ignored
	if n.Current() != 2 {
		t.Errorf("an out-of-range SetCurrent moved to %d", n.Current())
	}
	n.Remove(0)
	if n.Len() != 2 || len(reported) != 1 {
		t.Errorf("after Remove(0): %d pages, reported %v", n.Len(), reported)
	}
	n.Remove(5) // ignored
	if n.Len() != 2 {
		t.Errorf("an out-of-range Remove left %d pages", n.Len())
	}
	n.SetTabs([]string{"A"})
	if n.Len() != 1 {
		t.Errorf("after SetTabs: %d pages, want 1", n.Len())
	}
}
