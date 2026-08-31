// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package gtk4

import "errors"

// ErrUnsupported is returned by [Init] on any platform without GTK4 (everything
// but Linux). It is defined on every platform so a portable consumer can test
// for it with errors.Is.
var ErrUnsupported = errors.New("gtk4: only available on Linux")
