// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build !(linux && !android && (amd64 || arm64))

package gtk4

// The collection widgets do nothing off Linux (and off amd64/arm64, where the
// binding dlopens libgtk-4 and the Linux build is limited to those two): the
// constructors return nil, as they do when GTK4 cannot be loaded, and the methods
// are no-ops or zero, so consumer code that names them still compiles.

// ColumnView is a table. Off Linux none is ever created.
type ColumnView struct{}

func ColumnViewNew([]string) *ColumnView { return nil }
func (v *ColumnView) Widget() Widget     { return 0 }
func (v *ColumnView) Class() string      { return "" }
func (v *ColumnView) SetRows([][]string) {}
func (v *ColumnView) Len() int           { return 0 }
func (v *ColumnView) Rebuilds() int      { return 0 }
func (v *ColumnView) Selected() int      { return -1 }
func (v *ColumnView) SetSelected(int)    {}
func (v *ColumnView) Select(int)         {}
func (v *ColumnView) OnSelect(func(int)) {}
func (v *ColumnView) Destroy()           {}

// TreeView is a tree. Off Linux none is ever created.
type TreeView struct{}

func TreeViewNew() *TreeView              { return nil }
func (t *TreeView) Widget() Widget        { return 0 }
func (t *TreeView) Class() string         { return "" }
func (t *TreeView) SetNodes([]TreeNode)   {}
func (t *TreeView) Rebuilds() int         { return 0 }
func (t *TreeView) SetSelectedPath([]int) {}
func (t *TreeView) SelectPath([]int)      {}
func (t *TreeView) SelectedPath() []int   { return nil }
func (t *TreeView) OnSelect(func([]int))  {}
func (t *TreeView) Destroy()              {}

// Notebook is a tab strip. Off Linux none is ever created.
type Notebook struct{}

func NotebookNew() *Notebook           { return nil }
func (n *Notebook) Widget() Widget     { return 0 }
func (n *Notebook) Class() string      { return "" }
func (n *Notebook) SetTabs([]string)   {}
func (n *Notebook) Len() int           { return 0 }
func (n *Notebook) Current() int       { return -1 }
func (n *Notebook) SetCurrent(int)     {}
func (n *Notebook) Remove(int)         {}
func (n *Notebook) Switch(int)         {}
func (n *Notebook) OnSwitch(func(int)) {}
func (n *Notebook) Destroy()           {}
