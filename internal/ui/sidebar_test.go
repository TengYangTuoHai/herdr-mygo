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

// TestSidebarRowsCutLongTextOneLineHigh draws a space's name and its git
// chip in a row too narrow for them: both end in an ellipsis on one line
// instead of wrapping into a second, which overflows their 26 px row.
func TestSidebarRowsCutLongTextOneLineHigh(t *testing.T) {
	RegisterFonts()
	app := NewApp(&herdr.Session{}, ThemeByName("graphite"))
	name := "a-space-name-longer-than-the-sidebar-shows"
	branch := "feature/a-branch-name-longer-than-the-sidebar-shows"
	cases := []struct {
		what  string
		width float32
		text  string
		draw  func(c *myui.Context)
	}{
		{"a space's name", 90, name, func(c *myui.Context) {
			sp := &herdr.Space{ID: "sp1", Name: name}
			app.spaceRow(c, sp, []*herdr.Space{sp}, nil)
		}},
		{"a git chip's branch", 100, branch, func(c *myui.Context) {
			app.gitChip(c, branch)
		}},
	}
	for _, c := range cases {
		draw := c.draw
		tt := myui.NewTester(func(ctx *myui.Context) {
			myui.Column(ctx).Width(c.width).Children(func() { draw(ctx) })
		}, 400, 200)
		tt.Frame()
		r, ok := tt.Find(c.text)
		if !ok {
			t.Fatalf("%s: %q is not drawn", c.what, c.text)
		}
		if r.H > 20 {
			t.Errorf("%s is %v high at a %v wide row, so it wrapped: %v", c.what, r.H, c.width, r)
		}
	}
}

// TestAgentRowsLineUpWithSpaceRows keeps an agent's row the height of a
// space's, and its text starting where the space name does, so the two lists
// share one surface and one hover.
func TestAgentRowsLineUpWithSpaceRows(t *testing.T) {
	RegisterFonts()
	app := NewApp(&herdr.Session{}, ThemeByName("graphite"))
	sp := &herdr.Space{ID: "sp1", Name: "a-space"}
	ag := &herdr.Agent{SpaceName: "a-space", TabName: "a-tab", Status: "idle"}
	tt := myui.NewTester(func(c *myui.Context) {
		myui.Column(c).Width(268).PaddingX(8).Gap(2).Children(func() {
			app.spaceRow(c, sp, []*herdr.Space{sp}, nil)
			app.agentRow(c, ag)
		})
	}, 400, 200)
	tt.Frame()
	space, ok := tt.Find("a-space")
	if !ok {
		t.Fatal("the space row is not drawn")
	}
	agent, ok := tt.Find("a-space · a-tab")
	if !ok {
		t.Fatal("the agent row is not drawn")
	}
	if got := agent.Y - space.Y; got != 28 {
		t.Errorf("the agent row starts %v below the space row, want 28: 26 px rows with the 2 px gap", got)
	}
	if got := agent.X - space.X; got != 0 {
		t.Errorf("the agent's name starts %v past the space's, want them in one column", got)
	}
}
