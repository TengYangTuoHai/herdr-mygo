package ui

import (
	"os"
	"path/filepath"
	"testing"

	myui "github.com/egoist/mygo/ui"
)

// Clicking the tab's cross must use the same close flow as its menu:
// remove only that tab, remove an empty space, and confirm running programs.
func TestTabCloseButton(t *testing.T) {
	RegisterFonts()
	for _, tc := range []struct {
		name string
		last bool
		busy bool
	}{
		{name: "last-tab", last: true},
		{name: "multiple-tabs"},
		{name: "running-program", last: true, busy: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.busy {
				t.Setenv("TAB_CLOSE_TEST_BUSY", "1")
			}
			s := fullscreenSession(t)
			for _, tab := range s.Places()[0].Tabs {
				if tab.ID != "full" && (tc.last || tab.ID != "split") {
					s.CloseTab(tab.ID)
				}
			}
			s.SelectTab("full")
			app := NewApp(s, ThemeByName("graphite"))
			t.Cleanup(func() {
				app.mu.Lock()
				defer app.mu.Unlock()
				for _, entry := range app.terms {
					_ = entry.term.Close()
				}
			})
			tt := myui.NewTester(app.View, 1000, 600)
			tt.SetFocused(true)
			tt.Frame()
			save := func(stage string) {
				t.Helper()
				if dir := os.Getenv("TAB_CLOSE_EVIDENCE"); dir != "" {
					writeFullscreenShot(t, tt, filepath.Join(dir, tc.name+"-"+stage+".png"))
				}
			}
			clickClose := func() {
				t.Helper()
				if r, ok := tt.Find("Close tab"); ok {
					tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
					return
				}
				// Reproduce the original decorative cross before it has a label.
				r, ok := tt.Find("Full")
				if !ok {
					t.Fatal("active tab is not rendered")
				}
				tt.ClickAt(r.X+r.W+13, r.Y+r.H/2)
			}
			save("before")
			// The label still selects the tab instead of closing it.
			if err := tt.Click("Full"); err != nil {
				t.Fatal(err)
			}
			if _, tab := s.Active(); tab == nil || tab.ID != "full" {
				t.Fatal("clicking the tab label closed it")
			}
			clickClose()
			tt.Frame()
			save("clicked")
			if tc.busy {
				if !tt.HasText("Close tab?") || !tt.HasText("sleep") {
					t.Fatal("closing a running program did not ask for confirmation")
				}
				if s.SpaceOf("full") == nil {
					t.Fatal("tab closed before confirmation")
				}
				if err := tt.Click("Cancel"); err != nil {
					t.Fatal(err)
				}
				if s.SpaceOf("full") == nil {
					t.Fatal("cancel closed the tab")
				}
				clickClose()
				// Cancel has initial focus; Tab reaches the dialog's confirm
				// button without matching the toolbar's identical label.
				tt.Key(0, myui.KeyTab)
				tt.Key(0, myui.KeyEnter)
				tt.Frame()
			}
			save("after")
			if s.SpaceOf("full") != nil || tt.HasText("Full") {
				t.Fatal("clicking the cross did not close the tab")
			}
			if tc.last {
				if len(s.Places()) != 0 {
					t.Fatal("closing the last tab left its empty space")
				}
			} else {
				sp, tab := s.Active()
				if sp == nil || sp.ID != "space" || tab == nil || tab.ID != "split" || len(sp.Tabs) != 1 || !tt.HasText("Split") {
					t.Fatal("closing one tab did not preserve and select the other tab")
				}
			}
		})
	}
}
