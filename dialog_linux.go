// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build linux && !android && (amd64 || arm64)

package gtk4

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// minDialogMinor is the first GTK 4 minor release with GtkFileDialog and
// GtkAlertDialog: every method of both is "Available since: 4.10"
// (https://docs.gtk.org/gtk4/class.FileDialog.html,
// https://docs.gtk.org/gtk4/class.AlertDialog.html).
const minDialogMinor = 10

var (
	dlgOnce sync.Once
	dlgOK   bool
	dlgErr  error

	dlgMinor func() uint32
	dlgMajor func() uint32

	fdNew              func() uintptr
	fdSetTitle         func(uintptr, string)
	fdSetInitialName   func(uintptr, string)
	fdSetFilters       func(uintptr, uintptr)
	fdSetDefaultFilter func(uintptr, uintptr)
	fdAsync            [4]func(self, parent, cancellable, cb, data uintptr)
	fdFinish           [4]func(self, res, errp uintptr) uintptr

	ffNew     func() uintptr
	ffType    func() uint64
	ffSetName func(uintptr, string)
	ffSuffix  func(uintptr, string)

	lsNew    func(uint64) uintptr
	lsAppend func(uintptr, uintptr)
	lmCount  func(uintptr) uint32
	lmItem   func(uintptr, uint32) uintptr

	adType          func() uint64
	gObjectNewProps func(uint64, uint32, uintptr, uintptr) uintptr
	adSetMessage    func(uintptr, string)
	adSetDetail     func(uintptr, string)
	adSetButtons    func(uintptr, uintptr)
	adSetModal      func(uintptr, bool)
	adSetDefault    func(uintptr, int32)
	adChoose        func(self, parent, cancellable, cb, data uintptr)
	adChooseFinish  func(self, res, errp uintptr) int32

	gFileGetPath      func(uintptr) uintptr
	dlgFree           func(uintptr)
	gErrorFree        func(uintptr)
	gtkDialogQuark    func() uint32
	gioErrorQuark     func() uint32
	dlgObjectUnref    func(uintptr)
	dlgIdleAdd        func(uintptr, uintptr) uint32
	dlgIdleCb         uintptr
	dlgDoneCb         uintptr
	dlgCallbacksReady sync.Once
)

func loadDialogs() error {
	dlgOnce.Do(func() {
		if dlgErr = load(); dlgErr != nil {
			return
		}
		gtk, err := dlopenAny("libgtk-4.so.1", "libgtk-4.so")
		if err != nil {
			dlgErr = err
			return
		}
		glib, err := dlopenAny("libglib-2.0.so.0", "libglib-2.0.so")
		if err != nil {
			dlgErr = err
			return
		}
		gobj, err := dlopenAny("libgobject-2.0.so.0", "libgobject-2.0.so")
		if err != nil {
			dlgErr = err
			return
		}
		gio, err := dlopenAny("libgio-2.0.so.0", "libgio-2.0.so")
		if err != nil {
			dlgErr = err
			return
		}
		reg := func(p any, h uintptr, name string) { purego.RegisterLibFunc(p, h, name) }
		reg(&dlgMinor, gtk, "gtk_get_minor_version")
		reg(&dlgMajor, gtk, "gtk_get_major_version")
		if dlgMajor() != 4 || int(dlgMinor()) < minDialogMinor {
			return // dlgOK stays false; the 4.10 symbols below do not exist
		}
		reg(&fdNew, gtk, "gtk_file_dialog_new")
		reg(&fdSetTitle, gtk, "gtk_file_dialog_set_title")
		reg(&fdSetInitialName, gtk, "gtk_file_dialog_set_initial_name")
		reg(&fdSetFilters, gtk, "gtk_file_dialog_set_filters")
		reg(&fdSetDefaultFilter, gtk, "gtk_file_dialog_set_default_filter")
		for i, n := range [4]string{"open", "open_multiple", "select_folder", "save"} {
			reg(&fdAsync[i], gtk, "gtk_file_dialog_"+n)
			reg(&fdFinish[i], gtk, "gtk_file_dialog_"+n+"_finish")
		}
		reg(&ffNew, gtk, "gtk_file_filter_new")
		reg(&ffType, gtk, "gtk_file_filter_get_type")
		reg(&ffSetName, gtk, "gtk_file_filter_set_name")
		reg(&ffSuffix, gtk, "gtk_file_filter_add_suffix")
		reg(&lsNew, gio, "g_list_store_new")
		reg(&lsAppend, gio, "g_list_store_append")
		reg(&lmCount, gio, "g_list_model_get_n_items")
		reg(&lmItem, gio, "g_list_model_get_item")
		reg(&adType, gtk, "gtk_alert_dialog_get_type")
		// g_object_new is variadic; the _with_properties form takes a count.
		reg(&gObjectNewProps, gobj, "g_object_new_with_properties")
		reg(&adSetMessage, gtk, "gtk_alert_dialog_set_message")
		reg(&adSetDetail, gtk, "gtk_alert_dialog_set_detail")
		reg(&adSetButtons, gtk, "gtk_alert_dialog_set_buttons")
		reg(&adSetModal, gtk, "gtk_alert_dialog_set_modal")
		reg(&adSetDefault, gtk, "gtk_alert_dialog_set_default_button")
		reg(&adChoose, gtk, "gtk_alert_dialog_choose")
		reg(&adChooseFinish, gtk, "gtk_alert_dialog_choose_finish")
		reg(&gFileGetPath, gio, "g_file_get_path")
		reg(&dlgFree, glib, "g_free")
		reg(&gErrorFree, glib, "g_error_free")
		reg(&gtkDialogQuark, gtk, "gtk_dialog_error_quark")
		reg(&gioErrorQuark, gio, "g_io_error_quark")
		reg(&dlgObjectUnref, gobj, "g_object_unref")
		reg(&dlgIdleAdd, glib, "g_idle_add")
		dlgOK = true
	})
	return dlgErr
}

// DialogsSupported reports whether this process can show GtkFileDialog and
// GtkAlertDialog: libgtk-4 loads and is 4.10 or newer.
func DialogsSupported() bool { return loadDialogs() == nil && dlgOK }

func dialogsReady() error {
	if err := loadDialogs(); err != nil {
		return fmt.Errorf("%w: %v", ErrDialogUnsupported, err)
	}
	if !dlgOK {
		return ErrDialogUnsupported
	}
	return nil
}

// ---- running things on the GTK thread ----

var postQ struct {
	sync.Mutex
	fns []func()
}

// postMain queues fn to run on the GLib main loop thread. One persistent idle
// callback drains the queue, so a caller that posts often does not spend one of
// purego's finite callback slots each time.
func postMain(fn func()) {
	callbacksInit()
	postQ.Lock()
	postQ.fns = append(postQ.fns, fn)
	postQ.Unlock()
	dlgIdleAdd(dlgIdleCb, 0)
}

type pending struct {
	kind   int // 0..3 a file dialog (index into fdFinish), 4 an alert
	dialog uintptr
	file   func(FileDialogResult)
	alert  func(AlertResult)
}

var dlgReg = struct {
	sync.Mutex
	next uintptr
	m    map[uintptr]*pending
}{m: map[uintptr]*pending{}}

func callbacksInit() {
	dlgCallbacksReady.Do(func() {
		dlgIdleCb = purego.NewCallback(func(_ uintptr) int32 {
			postQ.Lock()
			fns := postQ.fns
			postQ.fns = nil
			postQ.Unlock()
			for _, f := range fns {
				f()
			}
			return 0 // G_SOURCE_REMOVE
		})
		retainCallback(dlgIdleCb)
		// A GAsyncReadyCallback: (source object, GAsyncResult, user data).
		dlgDoneCb = purego.NewCallback(func(src, res, data uintptr) {
			dlgReg.Lock()
			p := dlgReg.m[data]
			delete(dlgReg.m, data)
			dlgReg.Unlock()
			if p == nil {
				return
			}
			p.finish(src, res)
		})
		retainCallback(dlgDoneCb)
	})
}

// gerror is a decoded GError: {GQuark domain; gint code; gchar *message}.
type gerror struct {
	dialogDomain bool
	ioDomain     bool
	code         int
	msg          string
}

func takeError(errp uintptr) *gerror {
	if errp == 0 {
		return nil
	}
	type raw struct {
		domain uint32
		code   int32
		msg    uintptr
	}
	r := (*raw)(*(*unsafe.Pointer)(unsafe.Pointer(&errp)))
	e := &gerror{
		dialogDomain: r.domain == gtkDialogQuark(),
		ioDomain:     r.domain == gioErrorQuark(),
		code:         int(r.code),
		msg:          cString(r.msg),
	}
	gErrorFree(errp)
	return e
}

func (e *gerror) cancelled() bool {
	return (e.dialogDomain && IsCancel(true, e.code)) || (e.ioDomain && IsCancel(false, e.code))
}

// takePath returns a GFile's local path and drops the file.
func takePath(file uintptr) (string, bool) {
	pp := gFileGetPath(file)
	dlgObjectUnref(file)
	if pp == 0 {
		return "", false
	}
	s := cString(pp)
	dlgFree(pp)
	return s, true
}

func (p *pending) finish(src, res uintptr) {
	defer dlgObjectUnref(src)
	if p.kind == 4 {
		var errp uintptr
		btn := adChooseFinish(src, res, uintptr(unsafe.Pointer(&errp)))
		r := AlertResult{Button: int(btn)}
		if e := takeError(errp); e != nil {
			if e.cancelled() {
				r.Cancelled = true
			} else {
				r.Err = errors.New("gtk4: alert: " + e.msg)
			}
		} else if btn < 0 {
			r.Cancelled = true // dismissed with no cancel button set
		}
		p.alert(r)
		return
	}
	var errp uintptr
	out := fdFinish[p.kind](src, res, uintptr(unsafe.Pointer(&errp)))
	r := FileDialogResult{}
	if e := takeError(errp); e != nil {
		if e.cancelled() {
			r.Cancelled = true
		} else {
			r.Err = errors.New("gtk4: file dialog: " + e.msg)
		}
		p.file(r)
		return
	}
	if out == 0 {
		r.Cancelled = true
		p.file(r)
		return
	}
	if FileDialogKind(p.kind) == FileOpenMultiple {
		for i, n := uint32(0), lmCount(out); i < n; i++ {
			path, ok := takePath(lmItem(out, i))
			if !ok {
				r.Err = errors.New("gtk4: file dialog: a chosen file has no local path")
				r.Paths = nil
				break
			}
			r.Paths = append(r.Paths, path)
		}
		dlgObjectUnref(out)
		p.file(r)
		return
	}
	path, ok := takePath(out)
	if !ok {
		r.Err = errors.New("gtk4: file dialog: the chosen file has no local path")
	} else {
		r.Paths = []string{path}
	}
	p.file(r)
}

func (p *pending) register() uintptr {
	dlgReg.Lock()
	dlgReg.next++
	id := dlgReg.next
	dlgReg.m[id] = p
	dlgReg.Unlock()
	return id
}

// Index of a FileDialogKind in fdAsync/fdFinish: open, open_multiple,
// select_folder, save.
func kindSlot(k FileDialogKind) int {
	switch k {
	case FileOpenMultiple:
		return 1
	case FileSelectFolder:
		return 2
	case FileSave:
		return 3
	}
	return 0
}

// ShowFileDialog shows a GtkFileDialog over parent (which may be the null
// widget) and calls done, on the GTK main loop thread, exactly once -- unless it
// returns an error, in which case done is never called. It may be called from any
// goroutine: the dialog is created on the main loop thread, so the loop must be
// running for it to appear.
func ShowFileDialog(parent Widget, req FileDialogRequest, done func(FileDialogResult)) error {
	if err := dialogsReady(); err != nil {
		return err
	}
	if done == nil {
		return errors.New("gtk4: nil dialog callback")
	}
	postMain(func() {
		d := fdNew()
		slot := kindSlot(req.Kind)
		p := &pending{kind: slot, dialog: d, file: done}
		if req.Title != "" {
			fdSetTitle(d, req.Title)
		}
		if req.Kind == FileSave && req.InitialName != "" {
			fdSetInitialName(d, req.InitialName)
		}
		if req.Kind != FileSelectFolder && len(req.Filters) > 0 {
			store := lsNew(ffType())
			var first uintptr
			for _, f := range req.Filters {
				ff := ffNew()
				ffSetName(ff, f.Name)
				for _, s := range f.Suffixes {
					ffSuffix(ff, s)
				}
				lsAppend(store, ff)
				if first == 0 {
					first = ff
				}
				dlgObjectUnref(ff) // the store holds it
			}
			fdSetFilters(d, store)
			// first is still alive: the dialog now holds the store, which holds it.
			fdSetDefaultFilter(d, first)
			dlgObjectUnref(store)
		}
		id := p.register()
		fdAsync[slot](d, uintptr(parent), 0, dlgDoneCb, id)
	})
	return nil
}

// ShowAlert shows a GtkAlertDialog over parent and calls done, on the GTK main
// loop thread, exactly once -- unless it returns an error. Buttons may be empty
// (GTK then shows its own "Close"). The first button is the default.
func ShowAlert(parent Widget, req AlertRequest, done func(AlertResult)) error {
	if err := dialogsReady(); err != nil {
		return err
	}
	if done == nil {
		return errors.New("gtk4: nil dialog callback")
	}
	postMain(func() {
		d := gObjectNewProps(adType(), 0, 0, 0)
		adSetMessage(d, req.Message)
		adSetDetail(d, req.Detail)
		adSetModal(d, parent != 0)
		if len(req.Buttons) > 0 {
			bufs := make([][]byte, len(req.Buttons))
			ptrs := make([]uintptr, len(req.Buttons)+1) // NULL-terminated
			for i, b := range req.Buttons {
				bufs[i] = append([]byte(b), 0)
				ptrs[i] = uintptr(unsafe.Pointer(&bufs[i][0]))
			}
			adSetButtons(d, uintptr(unsafe.Pointer(&ptrs[0])))
			adSetDefault(d, 0)
			runtime.KeepAlive(bufs)
			runtime.KeepAlive(ptrs)
		}
		p := &pending{kind: 4, dialog: d, alert: done}
		id := p.register()
		adChoose(d, uintptr(parent), 0, dlgDoneCb, id)
	})
	return nil
}
