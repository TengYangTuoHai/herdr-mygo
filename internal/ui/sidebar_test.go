package ui

import (
	"testing"
	"time"

	myui "github.com/egoist/mygo/ui"

	"herdr-mygo/internal/herdr"
)

// TestToggleSidebarCollapsesToNothingAndRestoresWidth closes the sidebar to
// no width at all, then opens it to the width it had.
func TestToggleSidebarCollapsesToNothingAndRestoresWidth(t *testing.T) {
	RegisterFonts()
	app := NewApp(&herdr.Session{}, ThemeByName("graphite"))
	app.SidebarWidth = 300
	tt := myui.NewTester(app.View, 1280, 820)
	tt.Frame()
	if got := app.sidebarShown; got < 299 || got > 301 {
		t.Fatalf("open sidebar width = %v, want 300", got)
	}

	app.toggleSidebar()
	tt.Frame()
	tt.Advance(250 * time.Millisecond)
	if got := app.sidebarShown; got > 0.5 {
		t.Fatalf("closed sidebar width = %v, want 0", got)
	}

	app.toggleSidebar()
	tt.Frame()
	tt.Advance(250 * time.Millisecond)
	if got := app.sidebarShown; got < 299 || got > 301 {
		t.Fatalf("reopened sidebar width = %v, want 300", got)
	}
}

// TestSidebarEdgeDragKeepsItsLimits drags the sidebar's edge past both ends:
// the width stays between the limits, and is the width the sidebar opens
// with again.
func TestSidebarEdgeDragKeepsItsLimits(t *testing.T) {
	RegisterFonts()
	app := NewApp(&herdr.Session{}, ThemeByName("graphite"))
	tt := myui.NewTester(app.View, 1280, 820)
	tt.Frame()

	start := app.sidebarShown - 2
	tt.Press(start, 200)
	tt.Frame()
	for i := 1; i <= 10; i++ {
		tt.Move(start-float32(i*40), 200)
		tt.Frame()
	}
	tt.Release(start-400, 200)
	tt.Frame()
	if got := app.SidebarWidth; got < sidebarMinWidth-0.01 || got > sidebarMinWidth+0.01 {
		t.Fatalf("dragged left to %v, want %v", got, sidebarMinWidth)
	}
	if got := app.settings.SidebarWidth; got < sidebarMinWidth-0.01 || got > sidebarMinWidth+0.01 {
		t.Fatalf("saved width = %v, want %v", got, sidebarMinWidth)
	}

	edge := app.sidebarShown - 2
	tt.Press(edge, 200)
	tt.Frame()
	for i := 1; i <= 20; i++ {
		tt.Move(edge+float32(i*40), 200)
		tt.Frame()
	}
	tt.Release(edge+800, 200)
	tt.Frame()
	if got := app.SidebarWidth; got < sidebarMaxWidth-0.01 || got > sidebarMaxWidth+0.01 {
		t.Fatalf("dragged right to %v, want %v", got, sidebarMaxWidth)
	}
}

// TestSidebarStatePersistsBetweenLaunches keeps the sidebar's width and
// whether it is open in the app's own file.
func TestSidebarStatePersistsBetweenLaunches(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	first := NewApp(nil, ThemeByName("graphite"))
	first.LoadSettings()
	first.SidebarWidth = 320
	first.toggleSidebar()
	first.saveSettings()

	second := NewApp(nil, ThemeByName("graphite"))
	second.LoadSettings()
	if second.SidebarOpen {
		t.Error("the sidebar came back open")
	}
	if second.SidebarWidth != 320 {
		t.Errorf("sidebar width = %v, want 320", second.SidebarWidth)
	}
}
