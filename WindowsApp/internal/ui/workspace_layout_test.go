package ui

import (
	"testing"

	"zhengshi-wms-windowsapp/internal/config"
)

func TestCloneWorkspaceLayoutsDoesNotAliasConfigState(t *testing.T) {
	tables := map[string]config.TableLayoutState{
		"material.results": {ColumnOrder: []string{"Name", "Model"}, ColumnWidths: map[string]int{"Name": 180}},
	}
	splitters := map[string][]int{"main.workspace": {176, 900}}

	tableCopy := cloneTableLayouts(tables)
	splitterCopy := cloneSplitterLayouts(splitters)
	tableCopy["material.results"].ColumnOrder[0] = "Changed"
	tableCopy["material.results"].ColumnWidths["Name"] = 400
	splitterCopy["main.workspace"][0] = 1

	if tables["material.results"].ColumnOrder[0] != "Name" || tables["material.results"].ColumnWidths["Name"] != 180 {
		t.Fatal("table layout clone aliases source state")
	}
	if splitters["main.workspace"][0] != 176 {
		t.Fatal("splitter layout clone aliases source state")
	}
}

func TestSanitizeColumnWidth(t *testing.T) {
	for _, test := range []struct {
		input int
		want  int
	}{{1, minimumColumnWidth}, {240, 240}, {5000, maximumColumnWidth}} {
		if got := sanitizeColumnWidth(test.input); got != test.want {
			t.Fatalf("sanitizeColumnWidth(%d) = %d, want %d", test.input, got, test.want)
		}
	}
}
