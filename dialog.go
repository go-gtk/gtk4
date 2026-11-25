// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package gtk4

import "errors"

// ErrDialogUnsupported is returned by the dialog functions when the GTK in use
// is older than 4.10, the release that added GtkFileDialog and GtkAlertDialog
// ("Available since: 4.10"), and on every platform but Linux.
var ErrDialogUnsupported = errors.New("gtk4: GtkFileDialog and GtkAlertDialog need GTK 4.10")

// FileDialogKind says which GtkFileDialog entry point a request uses.
type FileDialogKind int

const (
	FileOpen         FileDialogKind = iota // gtk_file_dialog_open: one existing file
	FileOpenMultiple                       // gtk_file_dialog_open_multiple
	FileSelectFolder                       // gtk_file_dialog_select_folder
	FileSave                               // gtk_file_dialog_save
)

// A FileFilter is a named set of file-name suffixes, without the dot. Matching is
// case-insensitive (gtk_file_filter_add_suffix).
type FileFilter struct {
	Name     string
	Suffixes []string
}

// FileDialogRequest configures a GtkFileDialog.
type FileDialogRequest struct {
	Kind        FileDialogKind
	Title       string
	InitialName string       // a suggested file name for FileSave
	Filters     []FileFilter // the first is the default
}

// FileDialogResult is what a file dialog answered. Paths are g_file_get_path's
// answer, untouched. Cancelled is true when the person dismissed the dialog.
type FileDialogResult struct {
	Paths     []string
	Cancelled bool
	Err       error
}

// AlertRequest configures a GtkAlertDialog. Buttons are shown in order; with no
// cancel button, Escape and the close control dismiss the alert.
type AlertRequest struct {
	Message string
	Detail  string
	Buttons []string
}

// AlertResult is what an alert answered: Button is the index of the button
// pressed. Cancelled is true when it was dismissed without one.
type AlertResult struct {
	Button    int
	Cancelled bool
	Err       error
}

// The GtkDialogError codes (https://docs.gtk.org/gtk4/error.DialogError.html) and
// G_IO_ERROR_CANCELLED, which a GCancellable raises.
const (
	dialogErrorCancelled = 1
	dialogErrorDismissed = 2
	ioErrorCancelled     = 19
)

// IsCancel reports whether a GError of the given domain kind and code is the
// person declining rather than a failure. dialogDomain is true for the
// GtkDialogError domain and false for the GIO one.
func IsCancel(dialogDomain bool, code int) bool {
	if dialogDomain {
		return code == dialogErrorCancelled || code == dialogErrorDismissed
	}
	return code == ioErrorCancelled
}
