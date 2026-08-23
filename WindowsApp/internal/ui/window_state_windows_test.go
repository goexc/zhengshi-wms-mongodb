package ui

import (
	"testing"

	"github.com/lxn/walk"
)

func TestClampWindowBoundsRestoresVisibleUsableWindow(t *testing.T) {
	desktop := walk.Rectangle{X: -1920, Y: 0, Width: 3840, Height: 1080}
	got := clampWindowBounds(walk.Rectangle{X: 3000, Y: -200, Width: 700, Height: 400}, desktop, 1024, 640)
	want := walk.Rectangle{X: 896, Y: 0, Width: 1024, Height: 640}
	if got != want {
		t.Fatalf("bounds = %#v, want %#v", got, want)
	}
}

func TestClampWindowBoundsFitsSmallDesktop(t *testing.T) {
	desktop := walk.Rectangle{X: 0, Y: 0, Width: 800, Height: 600}
	got := clampWindowBounds(walk.Rectangle{X: 10, Y: 10, Width: 1440, Height: 840}, desktop, 1024, 640)
	if got != desktop {
		t.Fatalf("bounds = %#v, want %#v", got, desktop)
	}
}
