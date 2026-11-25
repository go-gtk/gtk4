// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package gtk4

import "testing"

func TestIsCancel(t *testing.T) {
	for _, c := range []struct {
		dialog bool
		code   int
		want   bool
	}{
		{true, 0, false}, // GTK_DIALOG_ERROR_FAILED
		{true, 1, true},  // CANCELLED
		{true, 2, true},  // DISMISSED
		{true, 3, false},
		{true, 19, false},
		{false, 19, true}, // G_IO_ERROR_CANCELLED
		{false, 1, false},
		{false, 2, false},
	} {
		if got := IsCancel(c.dialog, c.code); got != c.want {
			t.Errorf("IsCancel(%v,%d) = %v, want %v", c.dialog, c.code, got, c.want)
		}
	}
}
