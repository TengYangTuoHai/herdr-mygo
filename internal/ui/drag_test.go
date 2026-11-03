package ui

import (
	"testing"

	myui "github.com/egoist/mygo/ui"

	"herdr-mygo/internal/herdr"
)

func TestDropIndexAt(t *testing.T) {
	// Rows are 26 tall: the top half drops before the target, the bottom
	// half after it, and a target below the dragged row moves up a place.
	cases := []struct {
		name     string
		y        float32
		from, to int
		want     int
	}{
		{"top half, moving up", 5, 3, 1, 1},
		{"bottom half, moving up", 21, 3, 1, 2},
		{"top half, moving down", 5, 1, 3, 2},
		{"bottom half, moving down", 21, 1, 3, 3},
		{"just above the middle", 12, 1, 2, 1}, // already there: no move
		{"just below the middle", 14, 1, 2, 2},
	}
	for _, c := range cases {
		if got := dropIndexAt(c.y, 26, c.from, c.to); got != c.want {
			t.Errorf("%s: dropIndexAt(%v, %d, %d) = %d, want %d", c.name, c.y, c.from, c.to, got, c.want)
		}
	}
}

// TestSplitHandleDragsTheDivider drags the line between the spaces and
// agents lists down the window: the spaces list grows by what the pointer
// travelled, not by its horizontal travel, which stays at zero.
func TestSplitHandleDragsTheDivider(t *testing.T) {
	RegisterFonts()
	app := NewApp(&herdr.Session{}, ThemeByName("graphite"))
	tt := myui.NewTester(app.View, 1280, 820)
	tt.Frame()

	listHeight := app.H - 46 - sidebarFooterHeight - 1
	start := 46 + listHeight*app.splitRatio
	before := listHeight * app.splitRatio

	tt.Press(120, start)
	tt.Frame()
	for i := 1; i <= 2; i++ {
		tt.Move(120, start+float32(i*20))
	}

	want := (before + 40) / listHeight
	if got := app.splitRatio; got < want-0.001 || got > want+0.001 {
		t.Fatalf("splitRatio = %v, want %v", got, want)
	}
}

// TestSplitHandleStopsAtItsLimits keeps the agents list showing, and the
// spaces list readable, however far the pointer goes.
func TestSplitHandleStopsAtItsLimits(t *testing.T) {
	RegisterFonts()
	app := NewApp(&herdr.Session{}, ThemeByName("graphite"))
	tt := myui.NewTester(app.View, 1280, 820)
	tt.Frame()

	listHeight := app.H - 46 - sidebarFooterHeight - 1
	start := 46 + listHeight*app.splitRatio

	tt.Press(120, start)
	tt.Frame()
	for i := 1; i <= 30; i++ {
		tt.Move(120, start+float32(i*40))
	}
	minSpaces, _ := splitLimits(listHeight)
	if got := listHeight * app.splitRatio; got < minSpaces-0.001 || got > listHeight-0.001 {
		t.Fatalf("spaces height = %v, outside %v..%v", got, minSpaces, listHeight)
	}
}
