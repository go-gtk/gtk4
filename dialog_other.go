// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

//go:build !(linux && !android && (amd64 || arm64))

package gtk4

// DialogsSupported is false off Linux.
func DialogsSupported() bool { return false }

// ShowFileDialog reports [ErrDialogUnsupported] off Linux; done is never called.
func ShowFileDialog(Widget, FileDialogRequest, func(FileDialogResult)) error {
	return ErrDialogUnsupported
}

// ShowAlert reports [ErrDialogUnsupported] off Linux; done is never called.
func ShowAlert(Widget, AlertRequest, func(AlertResult)) error { return ErrDialogUnsupported }
