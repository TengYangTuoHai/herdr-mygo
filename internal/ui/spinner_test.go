package ui

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestWorkingSpinnerTurns draws the status the agents list, the tabs and the
// pane headers show for a working agent: it must keep asking for frames while
// it shows, and its arc must have turned a quarter of a second later.
func TestWorkingSpinnerTurns(t *testing.T) {
	RegisterFonts()
	app := NewApp(nil, ThemeByName("graphite"))
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Background(app.T.Canvas).Center().Children(func() {
			app.agentStatus(c, "working")
		})
	}, 120, 120)
	tt.SetScale(2)
	tt.Frame()
	if !tt.FramePending() {
		t.Fatal("a working spinner asks for no frame, so it stands still")
	}
	before := tt.Image()
	tt.Advance(250 * time.Millisecond)
	after := tt.Image()
	if bytes.Equal(before.Pix, after.Pix) {
		t.Fatal("the working spinner did not turn")
	}
}
