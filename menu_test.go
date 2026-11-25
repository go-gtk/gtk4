// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package gtk4

import (
	"reflect"
	"testing"
)

func TestMenuSections(t *testing.T) {
	pick := func() {}
	cases := []struct {
		name  string
		items []MenuItem
		want  [][]MenuEntry
	}{
		{"empty", nil, nil},
		{"one", []MenuItem{{Label: "Open", OnPick: pick}},
			[][]MenuEntry{{{0, "Open", true}}}},
		{"separator splits", []MenuItem{{Label: "A", OnPick: pick}, {}, {Label: "B", OnPick: pick}},
			[][]MenuEntry{{{0, "A", true}}, {{2, "B", true}}}},
		{"edge and doubled separators make no empty section",
			[]MenuItem{{}, {}, {Label: "A", OnPick: pick}, {}, {}, {Label: "B", OnPick: pick}, {}},
			[][]MenuEntry{{{2, "A", true}}, {{5, "B", true}}}},
		{"only separators", []MenuItem{{}, {}}, nil},
		{"nil pick is inert", []MenuItem{{Label: "Remove"}},
			[][]MenuEntry{{{0, "Remove", false}}}},
		{"underscores doubled so GTK shows them", []MenuItem{{Label: "Save_as", OnPick: pick}},
			[][]MenuEntry{{{0, "Save__as", true}}}},
	}
	for _, c := range cases {
		if got := MenuSections(c.items); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}
