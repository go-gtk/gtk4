// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package gtk4

import (
	"strconv"
	"strings"
)

// The pure half of the collection widgets ([ColumnView], [TreeView], [Notebook]):
// how a row or a tree node is flattened into the one string a GtkStringObject
// holds, and read back. It has no GTK in it, so it builds and is unit-tested on
// every platform.
//
// A GtkStringList is the cheapest GListModel that can hold 100 000 items (one
// small GObject each, built by a single splice), and a GtkColumnView binds only
// the rows on screen, so an item carries a row's cells as one string and a column
// reads its own cell out at bind time.

// cellSep separates the cells of one row inside an item string. It is the ASCII
// unit separator, which a cell cannot contain: encodeRow drops it (and NUL, which
// would end the C string early) from every cell.
const cellSep = "\x1f"

// cleanCell removes the two bytes an item string cannot carry.
func cleanCell(s string) string {
	return strings.Map(func(r rune) rune {
		if r == 0 || r == 0x1f {
			return -1
		}
		return r
	}, s)
}

// encodeRow flattens the first ncols cells of a row into one item string; a short
// row is padded with empty cells so every column finds its index.
func encodeRow(cells []string, ncols int) string {
	var b strings.Builder
	for i := 0; i < ncols; i++ {
		if i > 0 {
			b.WriteString(cellSep)
		}
		if i < len(cells) {
			b.WriteString(cleanCell(cells[i]))
		}
	}
	return b.String()
}

// cellAt reads cell idx of an encoded row, "" when the row has fewer cells.
func cellAt(row string, idx int) string {
	for i := 0; i < idx; i++ {
		j := strings.Index(row, cellSep)
		if j < 0 {
			return ""
		}
		row = row[j+len(cellSep):]
	}
	if j := strings.Index(row, cellSep); j >= 0 {
		return row[:j]
	}
	return row
}

// TreeNode is one node of a [TreeView]: a title, its children, and whether it
// starts expanded.
type TreeNode struct {
	Title    string
	Children []TreeNode
	Expanded bool
}

// pathString is the "2.0.1" form of a node path.
func pathString(path []int) string {
	var b strings.Builder
	for i, p := range path {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(strconv.Itoa(p))
	}
	return b.String()
}

// parsePath is pathString's inverse; ok is false for anything that is not
// dot-separated non-negative integers (the empty string is the empty path).
func parsePath(s string) (path []int, ok bool) {
	if s == "" {
		return nil, true
	}
	for _, part := range strings.Split(s, ".") {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return nil, false
		}
		path = append(path, n)
	}
	return path, true
}

// treeItem encodes one tree node as "path SEP title".
func treeItem(path []int, title string) string {
	return pathString(path) + cellSep + cleanCell(title)
}

// splitTreeItem is treeItem's inverse.
func splitTreeItem(item string) (path, title string) {
	if j := strings.Index(item, cellSep); j >= 0 {
		return item[:j], item[j+len(cellSep):]
	}
	return item, ""
}

// treeNodeAt returns the node a path names, or nil when it names nothing.
func treeNodeAt(tree []TreeNode, path []int) *TreeNode {
	var n *TreeNode
	level := tree
	for _, i := range path {
		if i < 0 || i >= len(level) {
			return nil
		}
		n = &level[i]
		level = n.Children
	}
	return n
}

// childItems encodes the children of the node at path (nil path: the top level).
func childItems(tree []TreeNode, path []int) []string {
	level := tree
	if len(path) > 0 {
		n := treeNodeAt(tree, path)
		if n == nil {
			return nil
		}
		level = n.Children
	}
	out := make([]string, len(level))
	for i, c := range level {
		p := make([]int, len(path)+1)
		copy(p, path)
		p[len(path)] = i
		out[i] = treeItem(p, c.Title)
	}
	return out
}
