package ui

import (
	"testing"

	"github.com/lxn/win"
)

func TestClosableTabTitleAndHitTarget(t *testing.T) {
	if got := closableTabTitle("物料查询"); got != "物料查询  ×" {
		t.Fatalf("closableTabTitle() = %q", got)
	}

	rect := win.RECT{Left: 10, Top: 4, Right: 130, Bottom: 30}
	if !isTabClosePoint(rect, 110, 12, 26) {
		t.Fatal("expected point in the close target")
	}
	if isTabClosePoint(rect, 90, 12, 26) {
		t.Fatal("ordinary tab-title clicks must not close the tab")
	}
	if isTabClosePoint(rect, 120, 35, 26) {
		t.Fatal("points below the tab must not close the tab")
	}
}
