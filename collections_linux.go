// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// GTK4 collection widgets, bound over purego like the rest of the package (libgtk-4
// is dlopen'd, no cgo):
//
//   - [ColumnView]: a GtkColumnView in a GtkScrolledWindow, over a GtkSingleSelection
//     of a GtkStringList. The view binds only the rows on screen, so 10 000 rows
//     cost 10 000 small GObjects built by one gtk_string_list_splice, not 10 000
//     widgets.
//   - [TreeView]: a GtkListView over a GtkTreeListModel (GtkTreeExpander rows).
//     Children are made lazily, by the model's create_func, when a row expands.
//   - [Notebook]: a GtkNotebook whose pages are empty boxes: the tab strip only,
//     the host lays the page content out.
//
// Docs read (docs.gtk.org/gtk4): GtkColumnView, GtkColumnViewColumn,
// GtkTreeListModel, GtkTreeListRow, GtkTreeExpander, GtkSignalListItemFactory,
// GtkSingleSelection, GtkStringList, GtkNotebook.

var (
	collOnce sync.Once
	collErr  error

	cListModelNItems func(uintptr) uint32
	cObjectUnref     func(uintptr)
	cTypeNameOf      func(uintptr) string
	cConnect         func(uintptr, string, uintptr, uintptr, uintptr, int32) uint64

	cStrListNew    func(uintptr) uintptr
	cStrListSplice func(uintptr, uint32, uint32, uintptr)
	cStrObjString  func(uintptr) string

	cSelNew          func(uintptr) uintptr
	cSelAutoselect   func(uintptr, bool)
	cSelCanUnselect  func(uintptr, bool)
	cSelGetSelected  func(uintptr) uint32
	cSelSetSelected  func(uintptr, uint32)
	cSelSelectedItem func(uintptr) uintptr
	cFactoryNew      func() uintptr
	cItemSetChild    func(uintptr, uintptr)
	cItemGetChild    func(uintptr) uintptr
	cItemGetItem     func(uintptr) uintptr
	cLabelNew        func(string) uintptr
	cLabelSetText    func(uintptr, string)
	cWidgetHalign    func(uintptr, int32)
	cColViewNew      func(uintptr) uintptr
	cColViewAppend   func(uintptr, uintptr)
	cColNew          func(string, uintptr) uintptr
	cColResizable    func(uintptr, bool)
	cColExpand       func(uintptr, bool)
	cScrolledNew     func() uintptr
	cScrolledSetChld func(uintptr, uintptr)
	cListViewNew     func(uintptr, uintptr) uintptr
	cTreeModelNew    func(uintptr, bool, bool, uintptr, uintptr, uintptr) uintptr
	cTreeModelGetRow func(uintptr, uint32) uintptr
	cRowSetExpanded  func(uintptr, bool)
	cRowGetItem      func(uintptr) uintptr
	cExpanderNew     func() uintptr
	cExpanderSetChld func(uintptr, uintptr)
	cExpanderGetChld func(uintptr) uintptr
	cExpanderSetRow  func(uintptr, uintptr)
	cNotebookNew     func() uintptr
	cNotebookAppend  func(uintptr, uintptr, uintptr) int32
	cNotebookRemove  func(uintptr, int32)
	cNotebookNPages  func(uintptr) int32
	cNotebookCurrent func(uintptr) int32
	cNotebookSetCur  func(uintptr, int32)
	cBoxNew          func(int32, int32) uintptr

	cbSetupLabel    uintptr
	cbBindCell      uintptr
	cbSetupExpander uintptr
	cbBindNode      uintptr
	cbSelected      uintptr
	cbChildren      uintptr
	cbSwitchPage    uintptr

	collMu  sync.Mutex
	collSeq uintptr
	collReg = map[uintptr]any{}
)

const invalidPos = 0xffffffff // GTK_INVALID_LIST_POSITION

func loadCollections() error {
	collOnce.Do(func() {
		if collErr = load(); collErr != nil {
			return
		}
		gio, err := dlopenAny("libgio-2.0.so.0", "libgio-2.0.so")
		if err != nil {
			collErr = err
			return
		}
		gobj, err := dlopenAny("libgobject-2.0.so.0", "libgobject-2.0.so")
		if err != nil {
			collErr = err
			return
		}
		gtk, err := dlopenAny("libgtk-4.so.1", "libgtk-4.so")
		if err != nil {
			collErr = err
			return
		}
		r := func(p any, h uintptr, name string) { purego.RegisterLibFunc(p, h, name) }
		r(&cListModelNItems, gio, "g_list_model_get_n_items")
		r(&cObjectUnref, gobj, "g_object_unref")
		r(&cTypeNameOf, gobj, "g_type_name_from_instance")
		r(&cConnect, gobj, "g_signal_connect_data")
		r(&cStrListNew, gtk, "gtk_string_list_new")
		r(&cStrListSplice, gtk, "gtk_string_list_splice")
		r(&cStrObjString, gtk, "gtk_string_object_get_string")
		r(&cSelNew, gtk, "gtk_single_selection_new")
		r(&cSelAutoselect, gtk, "gtk_single_selection_set_autoselect")
		r(&cSelCanUnselect, gtk, "gtk_single_selection_set_can_unselect")
		r(&cSelGetSelected, gtk, "gtk_single_selection_get_selected")
		r(&cSelSetSelected, gtk, "gtk_single_selection_set_selected")
		r(&cSelSelectedItem, gtk, "gtk_single_selection_get_selected_item")
		r(&cFactoryNew, gtk, "gtk_signal_list_item_factory_new")
		r(&cItemSetChild, gtk, "gtk_list_item_set_child")
		r(&cItemGetChild, gtk, "gtk_list_item_get_child")
		r(&cItemGetItem, gtk, "gtk_list_item_get_item")
		r(&cLabelNew, gtk, "gtk_label_new")
		r(&cLabelSetText, gtk, "gtk_label_set_text")
		r(&cWidgetHalign, gtk, "gtk_widget_set_halign")
		r(&cColViewNew, gtk, "gtk_column_view_new")
		r(&cColViewAppend, gtk, "gtk_column_view_append_column")
		r(&cColNew, gtk, "gtk_column_view_column_new")
		r(&cColResizable, gtk, "gtk_column_view_column_set_resizable")
		r(&cColExpand, gtk, "gtk_column_view_column_set_expand")
		r(&cScrolledNew, gtk, "gtk_scrolled_window_new")
		r(&cScrolledSetChld, gtk, "gtk_scrolled_window_set_child")
		r(&cListViewNew, gtk, "gtk_list_view_new")
		r(&cTreeModelNew, gtk, "gtk_tree_list_model_new")
		r(&cTreeModelGetRow, gtk, "gtk_tree_list_model_get_row")
		r(&cRowSetExpanded, gtk, "gtk_tree_list_row_set_expanded")
		r(&cRowGetItem, gtk, "gtk_tree_list_row_get_item")
		r(&cExpanderNew, gtk, "gtk_tree_expander_new")
		r(&cExpanderSetChld, gtk, "gtk_tree_expander_set_child")
		r(&cExpanderGetChld, gtk, "gtk_tree_expander_get_child")
		r(&cExpanderSetRow, gtk, "gtk_tree_expander_set_list_row")
		r(&cNotebookNew, gtk, "gtk_notebook_new")
		r(&cNotebookAppend, gtk, "gtk_notebook_append_page")
		r(&cNotebookRemove, gtk, "gtk_notebook_remove_page")
		r(&cNotebookNPages, gtk, "gtk_notebook_get_n_pages")
		r(&cNotebookCurrent, gtk, "gtk_notebook_get_current_page")
		r(&cNotebookSetCur, gtk, "gtk_notebook_set_current_page")
		r(&cBoxNew, gtk, "gtk_box_new")

		// One callback per signal shape for the whole process (purego never frees a
		// callback and caps their number); the user-data word, a registry id, says
		// which view or column a call is for.
		cbSetupLabel = purego.NewCallback(func(_, item, _ uintptr) {
			l := cLabelNew("")
			cWidgetHalign(l, 1) // GTK_ALIGN_START
			cItemSetChild(item, l)
		})
		cbBindCell = purego.NewCallback(func(_, item, data uintptr) {
			col, _ := collLookup(data).(*collColumn)
			if col == nil {
				return
			}
			obj := cItemGetItem(item)
			if obj == 0 {
				return
			}
			cLabelSetText(cItemGetChild(item), cellAt(cStrObjString(obj), col.idx))
		})
		cbSetupExpander = purego.NewCallback(func(_, item, _ uintptr) {
			e := cExpanderNew()
			l := cLabelNew("")
			cWidgetHalign(l, 1)
			cExpanderSetChld(e, l)
			cItemSetChild(item, e)
		})
		cbBindNode = purego.NewCallback(func(_, item, _ uintptr) {
			row := cItemGetItem(item) // a GtkTreeListRow (not passed through)
			if row == 0 {
				return
			}
			e := cItemGetChild(item)
			cExpanderSetRow(e, row)
			obj := cRowGetItem(row)
			if obj == 0 {
				return
			}
			_, title := splitTreeItem(cStrObjString(obj))
			cObjectUnref(obj)
			cLabelSetText(cExpanderGetChld(e), title)
		})
		cbSelected = purego.NewCallback(func(_, _, data uintptr) {
			switch v := collLookup(data).(type) {
			case *ColumnView:
				v.selectionChanged()
			case *TreeView:
				v.selectionChanged()
			}
		})
		cbChildren = purego.NewCallback(func(item, data uintptr) uintptr {
			if t, _ := collLookup(data).(*TreeView); t != nil {
				return t.makeChildren(item)
			}
			return 0
		})
		cbSwitchPage = purego.NewCallback(func(_, _, page, data uintptr) {
			if n, _ := collLookup(data).(*Notebook); n != nil {
				n.pageSwitched(int(int32(uint32(page))))
			}
		})
		for _, cb := range []uintptr{cbSetupLabel, cbBindCell, cbSetupExpander, cbBindNode, cbSelected, cbChildren, cbSwitchPage} {
			retainCallback(cb)
		}
	})
	return collErr
}

func collRegister(v any) uintptr {
	collMu.Lock()
	defer collMu.Unlock()
	collSeq++
	collReg[collSeq] = v
	return collSeq
}

func collLookup(id uintptr) any {
	collMu.Lock()
	defer collMu.Unlock()
	return collReg[id]
}

func collUnregister(id uintptr) {
	collMu.Lock()
	delete(collReg, id)
	collMu.Unlock()
}

// collColumn is the user data of one column's bind callback.
type collColumn struct{ idx int }

// spliceAll replaces the whole content of a GtkStringList with items, in one
// gtk_string_list_splice (the additions are a NULL-terminated char** built from a
// single buffer, kept alive across the call).
func spliceAll(list uintptr, items []string) {
	total := 0
	for _, s := range items {
		total += len(s) + 1
	}
	buf := make([]byte, 0, total)
	ptrs := make([]uintptr, len(items)+1)
	offs := make([]int, len(items))
	for i, s := range items {
		offs[i] = len(buf)
		buf = append(buf, s...)
		buf = append(buf, 0)
	}
	var add uintptr
	if len(items) > 0 {
		base := uintptr(unsafe.Pointer(&buf[0]))
		for i := range items {
			ptrs[i] = base + uintptr(offs[i])
		}
		add = uintptr(unsafe.Pointer(&ptrs[0]))
	}
	cStrListSplice(list, 0, cListModelNItems(list), add)
	runtime.KeepAlive(buf)
	runtime.KeepAlive(ptrs)
}

func newSelection(model uintptr, id uintptr) uintptr {
	sel := cSelNew(model)
	cSelAutoselect(sel, false)
	cSelCanUnselect(sel, true)
	cConnect(sel, "notify::selected", cbSelected, id, 0, 0)
	return sel
}

func scrolledAround(view uintptr) uintptr {
	root := cScrolledNew()
	cScrolledSetChld(root, view)
	return root
}

// ---- ColumnView ----

// A ColumnView is a table: a GtkColumnView in a scroller, one string cell per
// column, one row selectable. The rows live in a GtkStringList and the view binds
// only those on screen, so [ColumnView.SetRows] on 100 000 rows is one splice, not
// 100 000 widgets.
type ColumnView struct {
	id    uintptr
	root  uintptr
	view  uintptr
	list  uintptr
	sel   uintptr
	ncols int
	ids   []uintptr

	mute     int // >0 while the host itself changes the selection
	rows     int
	rebuilds int
	onSelect func(int)
}

// ColumnViewNew builds a table with one column per title (resizable, expanding).
// It returns nil when GTK4 cannot be bound.
func ColumnViewNew(columns []string) *ColumnView {
	if loadCollections() != nil {
		return nil
	}
	v := &ColumnView{ncols: len(columns)}
	v.id = collRegister(v)
	v.list = cStrListNew(0)
	v.sel = newSelection(v.list, v.id) // the selection owns the list from here
	v.view = cColViewNew(v.sel)
	for i, title := range columns {
		f := cFactoryNew()
		cid := collRegister(&collColumn{idx: i})
		v.ids = append(v.ids, cid)
		cConnect(f, "setup", cbSetupLabel, 0, 0, 0)
		cConnect(f, "bind", cbBindCell, cid, 0, 0)
		col := cColNew(cleanCell(title), f) // takes the factory
		cColResizable(col, true)
		cColExpand(col, true)
		cColViewAppend(v.view, col)
		cObjectUnref(col) // the view holds its own reference
	}
	v.root = scrolledAround(v.view)
	return v
}

// Widget is the widget to place: the scroller around the view.
func (v *ColumnView) Widget() Widget { return Widget(v.root) }

// Class is the GTK class name of the widget that does the work ("GtkColumnView").
func (v *ColumnView) Class() string { return cTypeNameOf(v.view) }

// SetRows replaces every row with one splice (a single items-changed); the
// selection is cleared by GTK when the rows it pointed at go. A cell may not hold
// NUL or the ASCII unit separator: both are dropped.
func (v *ColumnView) SetRows(rows [][]string) {
	items := make([]string, len(rows))
	for i, r := range rows {
		items[i] = encodeRow(r, v.ncols)
	}
	v.mute++
	spliceAll(v.list, items)
	v.mute--
	v.rows = len(rows)
	v.rebuilds++
}

// Len is the number of rows.
func (v *ColumnView) Len() int { return v.rows }

// Rebuilds counts how many times SetRows replaced the data; a host that skips an
// unchanged table (and a test that proves it) reads it.
func (v *ColumnView) Rebuilds() int { return v.rebuilds }

// Selected is the selected row, or -1.
func (v *ColumnView) Selected() int {
	if p := cSelGetSelected(v.sel); p != invalidPos {
		return int(p)
	}
	return -1
}

// SetSelected selects row i, or clears the selection for i < 0 or out of range.
// It does not call the [ColumnView.OnSelect] handler: only the user does.
func (v *ColumnView) SetSelected(i int) {
	pos := uint32(invalidPos)
	if i >= 0 && i < v.rows {
		pos = uint32(i)
	}
	v.mute++
	cSelSetSelected(v.sel, pos)
	v.mute--
}

// Select is [ColumnView.SetSelected] as the person would do it: the
// [ColumnView.OnSelect] handler is called when the selection changes.
func (v *ColumnView) Select(i int) {
	pos := uint32(invalidPos)
	if i >= 0 && i < v.rows {
		pos = uint32(i)
	}
	cSelSetSelected(v.sel, pos)
}

// OnSelect sets the handler called with the new selected row (-1 when cleared)
// when the user (or [ColumnView.Select]) changes the selection.
func (v *ColumnView) OnSelect(fn func(int)) { v.onSelect = fn }

func (v *ColumnView) selectionChanged() {
	if v.mute > 0 || v.onSelect == nil {
		return
	}
	v.onSelect(v.Selected())
}

// Destroy releases the view's callbacks registry entries. The widget itself is
// owned by its parent, as for every other widget in this package.
func (v *ColumnView) Destroy() {
	collUnregister(v.id)
	for _, id := range v.ids {
		collUnregister(id)
	}
	v.ids = nil
}

// ---- TreeView ----

// A TreeView is a tree: a GtkListView over a GtkTreeListModel with a
// GtkTreeExpander per row. Children are made lazily, when a row expands.
type TreeView struct {
	id    uintptr
	root  uintptr
	view  uintptr
	list  uintptr // the GtkStringList of the top-level nodes
	tlm   uintptr // the GtkTreeListModel
	sel   uintptr
	nodes []TreeNode

	mute     int
	rebuilds int
	onSelect func([]int)
}

// TreeViewNew builds an empty tree. It returns nil when GTK4 cannot be bound.
func TreeViewNew() *TreeView {
	if loadCollections() != nil {
		return nil
	}
	t := &TreeView{}
	t.id = collRegister(t)
	t.list = cStrListNew(0)
	t.tlm = cTreeModelNew(t.list, false, false, cbChildren, t.id, 0) // takes the list
	t.sel = newSelection(t.tlm, t.id)
	f := cFactoryNew()
	cConnect(f, "setup", cbSetupExpander, 0, 0, 0)
	cConnect(f, "bind", cbBindNode, 0, 0, 0)
	t.view = cListViewNew(t.sel, f)
	t.root = scrolledAround(t.view)
	return t
}

// Widget is the widget to place: the scroller around the view.
func (t *TreeView) Widget() Widget { return Widget(t.root) }

// Class is the GTK class name of the widget that does the work ("GtkListView").
func (t *TreeView) Class() string { return cTypeNameOf(t.view) }

// makeChildren is the model's create_func: the children of the node an item
// stands for, as a new model, or 0 for a leaf.
func (t *TreeView) makeChildren(item uintptr) uintptr {
	path, _ := splitTreeItem(cStrObjString(item))
	idx, ok := parsePath(path)
	if !ok {
		return 0
	}
	kids := childItems(t.nodes, idx)
	if len(kids) == 0 {
		return 0
	}
	l := cStrListNew(0)
	spliceAll(l, kids)
	return l
}

// SetNodes replaces the tree and re-expands the nodes marked Expanded. Children
// are made as each row expands, so a large collapsed tree costs its top level.
func (t *TreeView) SetNodes(nodes []TreeNode) {
	t.mute++
	defer func() { t.mute-- }()
	t.nodes = nodes
	spliceAll(t.list, childItems(nodes, nil))
	// Expanding a row inserts its children right after it, so walking forward
	// reaches them next, and the walk is bounded by the clamped node count.
	for i := uint32(0); i < cListModelNItems(t.tlm); i++ {
		row := cTreeModelGetRow(t.tlm, i)
		if row == 0 {
			continue
		}
		if p, ok := t.rowPath(row); ok {
			if n := treeNodeAt(nodes, p); n != nil && n.Expanded && len(n.Children) > 0 {
				cRowSetExpanded(row, true)
			}
		}
		cObjectUnref(row)
	}
	t.rebuilds++
}

// Rebuilds counts how many times SetNodes replaced the tree.
func (t *TreeView) Rebuilds() int { return t.rebuilds }

// rowPath is the path of the node a GtkTreeListRow stands for.
func (t *TreeView) rowPath(row uintptr) ([]int, bool) {
	obj := cRowGetItem(row)
	if obj == 0 {
		return nil, false
	}
	path, _ := splitTreeItem(cStrObjString(obj))
	cObjectUnref(obj)
	return parsePath(path)
}

// findPath returns the position of the visible row for a path string, or -1.
func (t *TreeView) findPath(path string) int {
	n := cListModelNItems(t.tlm)
	for i := uint32(0); i < n; i++ {
		row := cTreeModelGetRow(t.tlm, i)
		if row == 0 {
			continue
		}
		obj := cRowGetItem(row)
		var p string
		if obj != 0 {
			p, _ = splitTreeItem(cStrObjString(obj))
			cObjectUnref(obj)
		}
		cObjectUnref(row)
		if p == path {
			return int(i)
		}
	}
	return -1
}

// SetSelectedPath selects the node at a path (child indexes from the top level),
// expanding its ancestors so the row exists; an empty path or one that names no
// node clears the selection. It does not call the [TreeView.OnSelect] handler.
func (t *TreeView) SetSelectedPath(path []int) {
	t.mute++
	defer func() { t.mute-- }()
	t.selectPath(path, true)
}

// SelectPath is [TreeView.SetSelectedPath] as the person would do it: the
// [TreeView.OnSelect] handler is called when the selection changes (once, with
// the final path: opening the ancestors is not reported).
func (t *TreeView) SelectPath(path []int) { t.selectPath(path, false) }

func (t *TreeView) selectPath(path []int, silent bool) {
	if len(path) == 0 || treeNodeAt(t.nodes, path) == nil {
		cSelSetSelected(t.sel, invalidPos)
		return
	}
	if !silent {
		t.mute++ // opening a row can move the selection's position, not its node
	}
	for d := 1; d < len(path); d++ {
		pos := t.findPath(pathString(path[:d]))
		if pos < 0 {
			if !silent {
				t.mute--
			}
			return
		}
		if row := cTreeModelGetRow(t.tlm, uint32(pos)); row != 0 {
			cRowSetExpanded(row, true)
			cObjectUnref(row)
		}
	}
	if !silent {
		t.mute--
	}
	if pos := t.findPath(pathString(path)); pos >= 0 {
		cSelSetSelected(t.sel, uint32(pos))
	}
}

// SelectedPath is the path of the selected node, or nil.
func (t *TreeView) SelectedPath() []int {
	if cSelGetSelected(t.sel) == invalidPos {
		return nil
	}
	row := cSelSelectedItem(t.sel)
	if row == 0 {
		return nil
	}
	p, _ := t.rowPath(row) // the selection keeps its own reference to the row
	return p
}

// OnSelect sets the handler called with the new selected path (nil when cleared)
// when the user changes the selection.
func (t *TreeView) OnSelect(fn func([]int)) { t.onSelect = fn }

func (t *TreeView) selectionChanged() {
	if t.mute > 0 || t.onSelect == nil {
		return
	}
	t.onSelect(t.SelectedPath())
}

// Destroy releases the view's registry entry. The widget itself is owned by its
// parent.
func (t *TreeView) Destroy() { collUnregister(t.id) }

// ---- Notebook ----

// A Notebook is a GtkNotebook whose pages are empty boxes: the tab strip only.
type Notebook struct {
	id       uintptr
	nb       uintptr
	titles   []string
	mute     int
	onSwitch func(int)
}

// NotebookNew builds an empty tab strip. It returns nil when GTK4 cannot be
// bound.
func NotebookNew() *Notebook {
	if loadCollections() != nil {
		return nil
	}
	n := &Notebook{nb: cNotebookNew()}
	n.id = collRegister(n)
	cConnect(n.nb, "switch-page", cbSwitchPage, n.id, 0, 0)
	return n
}

// Widget is the notebook itself.
func (n *Notebook) Widget() Widget { return Widget(n.nb) }

// Class is the GTK class name ("GtkNotebook").
func (n *Notebook) Class() string { return cTypeNameOf(n.nb) }

// SetTabs replaces every page with one empty page per title. The current page
// is whatever GTK leaves; call [Notebook.SetCurrent] after.
func (n *Notebook) SetTabs(titles []string) {
	n.mute++
	for k := cNotebookNPages(n.nb); k > 0; k-- {
		cNotebookRemove(n.nb, -1)
	}
	for _, t := range titles {
		cNotebookAppend(n.nb, cBoxNew(0, 0), cLabelNew(cleanCell(t)))
	}
	n.mute--
	n.titles = titles
}

// Len is the number of pages.
func (n *Notebook) Len() int { return int(cNotebookNPages(n.nb)) }

// Current is the current page, or -1 when there is none.
func (n *Notebook) Current() int { return int(cNotebookCurrent(n.nb)) }

// SetCurrent shows page i; an index out of range is ignored. It does not call
// the [Notebook.OnSwitch] handler.
func (n *Notebook) SetCurrent(i int) {
	if i < 0 || i >= len(n.titles) {
		return
	}
	n.mute++
	cNotebookSetCur(n.nb, int32(i))
	n.mute--
}

// Remove removes page i; an index out of range is ignored.
func (n *Notebook) Remove(i int) {
	if i < 0 || i >= len(n.titles) {
		return
	}
	n.mute++
	cNotebookRemove(n.nb, int32(i))
	n.mute--
	n.titles = append(append([]string(nil), n.titles[:i]...), n.titles[i+1:]...)
}

// Switch is [Notebook.SetCurrent] as the person would do it: the
// [Notebook.OnSwitch] handler is called when the page changes.
func (n *Notebook) Switch(i int) {
	if i < 0 || i >= len(n.titles) {
		return
	}
	cNotebookSetCur(n.nb, int32(i))
}

// OnSwitch sets the handler called with the new page when the user (or
// [Notebook.Switch]) switches.
func (n *Notebook) OnSwitch(fn func(int)) { n.onSwitch = fn }

func (n *Notebook) pageSwitched(i int) {
	if n.mute > 0 || n.onSwitch == nil {
		return
	}
	n.onSwitch(i)
}

// Destroy releases the notebook's registry entry. The widget itself is owned by
// its parent.
func (n *Notebook) Destroy() { collUnregister(n.id) }
