// Copyright (c) the go-gtk authors. All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package gtk4

import "testing"

func TestPlanBar(t *testing.T) {
	f := func() {}
	items := []BarItem{
		{Label: "File", Sub: []BarItem{
			{Label: "New_file", OnActivate: f},
			{Separator: true}, {Separator: true},
			{Label: "Recent", Sub: []BarItem{{Label: "a", OnActivate: f}}},
			{Label: "Quit", OnActivate: f},
			{Separator: true},
		}},
		{Label: "Edit", Sub: []BarItem{{Label: "Copy"}}},
	}
	secs, leaves := planBar(items)
	if len(leaves) != 4 {
		t.Fatalf("leaves = %d, want 4", len(leaves))
	}
	if len(secs) != 1 || len(secs[0]) != 2 {
		t.Fatalf("top = %v, want one section of two menus", secs)
	}
	file := secs[0][0]
	if file.label != "File" || len(file.sub) != 2 {
		t.Fatalf("File = %+v, want 2 sections (separators collapsed, trailing dropped)", file)
	}
	if got := file.sub[0][0].label; got != "New__file" {
		t.Errorf("underscore label = %q", got)
	}
	if file.sub[1][0].leaf != -1 || file.sub[1][0].sub[0][0].leaf != 1 {
		t.Errorf("submenu numbering wrong: %+v", file.sub[1])
	}
	if file.sub[1][1].leaf != 2 {
		t.Errorf("Quit leaf = %d, want 2", file.sub[1][1].leaf)
	}
}
