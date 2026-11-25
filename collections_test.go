// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package gtk4

import (
	"reflect"
	"testing"
)

func TestEncodeRowAndCellAt(t *testing.T) {
	row := encodeRow([]string{"a", "b\x1fc", "d\x00e"}, 4)
	want := "a" + cellSep + "bc" + cellSep + "de" + cellSep
	if row != want {
		t.Fatalf("encodeRow = %q, want %q", row, want)
	}
	for i, w := range []string{"a", "bc", "de", "", ""} {
		if got := cellAt(row, i); got != w {
			t.Errorf("cellAt(%d) = %q, want %q", i, got, w)
		}
	}
	if got := encodeRow([]string{"x", "y", "z"}, 2); got != "x"+cellSep+"y" {
		t.Errorf("a row longer than the columns is cut: %q", got)
	}
	if got := encodeRow(nil, 0); got != "" {
		t.Errorf("no columns = %q", got)
	}
	for _, c := range []struct {
		row  string
		idx  int
		want string
	}{{"a", 0, "a"}, {"a", 1, ""}, {"a\x1fb", 1, "b"}, {"a\x1fb", 2, ""}, {"a\x1fb", 5, ""}, {"", 0, ""}, {"", 3, ""}, {"one", 1, ""}} {
		if got := cellAt(c.row, c.idx); got != c.want {
			t.Errorf("cellAt(%q,%d) = %q, want %q", c.row, c.idx, got, c.want)
		}
	}
	if got := childItems(nil, nil); len(got) != 0 {
		t.Errorf("an empty tree has items: %q", got)
	}
	if got := childItems([]TreeNode{{Title: "A"}}, []int{0}); len(got) != 0 {
		t.Errorf("a leaf has children: %q", got)
	}
	if pathString(nil) != "" {
		t.Error("the empty path must be the empty string")
	}
}

func TestPathRoundTrip(t *testing.T) {
	for _, p := range [][]int{{0}, {2, 0, 1}, {10, 3}} {
		got, ok := parsePath(pathString(p))
		if !ok || !reflect.DeepEqual(got, p) {
			t.Errorf("round trip of %v = %v, %v", p, got, ok)
		}
	}
	if got, ok := parsePath(""); !ok || got != nil {
		t.Errorf("empty path = %v, %v", got, ok)
	}
	for _, bad := range []string{"a", "1..2", "-1", "1.", ".1"} {
		if _, ok := parsePath(bad); ok {
			t.Errorf("parsePath(%q) accepted", bad)
		}
	}
}

func TestTreeItems(t *testing.T) {
	tree := []TreeNode{
		{Title: "a", Children: []TreeNode{{Title: "a0"}, {Title: "a1", Children: []TreeNode{{Title: "a10"}}}}},
		{Title: "b\x1f"},
	}
	top := childItems(tree, nil)
	if len(top) != 2 {
		t.Fatalf("top = %q", top)
	}
	if p, title := splitTreeItem(top[1]); p != "1" || title != "b" {
		t.Errorf("split(top[1]) = %q, %q", p, title)
	}
	kids := childItems(tree, []int{0})
	if len(kids) != 2 || kids[1] != treeItem([]int{0, 1}, "a1") {
		t.Errorf("kids = %q", kids)
	}
	if n := treeNodeAt(tree, []int{0, 1, 0}); n == nil || n.Title != "a10" {
		t.Errorf("treeNodeAt = %v", n)
	}
	if p, title := splitTreeItem("lonely"); p != "lonely" || title != "" {
		t.Errorf("split without a separator = %q, %q", p, title)
	}
}

// guarded runs f and reports, by assertion, a failure of the guard under test
// instead of letting it end the test binary.
func guarded(t *testing.T, what string, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("%s: crashed: %v", what, r)
		}
	}()
	f()
}

func TestTreeGuardsHoldOnBadPaths(t *testing.T) {
	tree := []TreeNode{{Title: "A", Children: []TreeNode{{Title: "A0"}}}}
	for _, bad := range [][]int{{1}, {-1}, {0, 1}, {0, -1}, {0, 0, 0}} {
		guarded(t, "treeNodeAt "+pathString(bad), func() {
			if n := treeNodeAt(tree, bad); n != nil {
				t.Errorf("treeNodeAt(%v) = %v, want nil", bad, n)
			}
		})
		guarded(t, "childItems "+pathString(bad), func() {
			if got := childItems(tree, bad); len(got) != 0 {
				t.Errorf("childItems(%v) = %q, want none", bad, got)
			}
		})
	}
}
