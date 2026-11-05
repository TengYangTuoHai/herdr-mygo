package ui

import (
	"fmt"
	"unicode/utf8"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// findState belongs to one terminal view; the query stays when the bar closes.
type findState struct {
	key   string
	term  *terminal.Terminal
	query string
	focus bool
}

// openFind searches the terminal in front, including a peeked sub-agent.
func (a *App) openFind() {
	if a.overlay == "settings" {
		a.openSettingsSearch()
		return
	}
	if a.overlay == "peek" {
		a.mu.Lock()
		st := a.peek
		a.mu.Unlock()
		if st.addr != "" {
			a.openFindFor("peek|"+st.addr, st.term)
		}
		return
	}
	if a.isOverlay() || a.S == nil {
		return
	}
	p := a.S.FocusedPane()
	if p.Terminal != "" {
		a.openFindFor(p.Pane+"|"+p.Terminal, a.terminalFor(p))
	}
}

func (a *App) openFindFor(key string, term *terminal.Terminal) {
	if a.find.term != nil && a.find.term != term {
		a.find.term.EndSearch()
	}
	a.find.key, a.find.term, a.find.focus = key, term, true
	a.Repaint()
}

func (a *App) findStep(back bool) {
	if a.find.key == "" || (a.overlay == "peek" && a.find.key != "peek|"+a.peek.addr) {
		a.openFind()
		return
	}
	if a.isOverlay() && a.overlay != "peek" {
		return
	}
	if term := a.find.term; term != nil {
		if back {
			term.SearchPrevious()
		} else {
			term.SearchNext()
		}
	}
}

// closeFind removes the highlights and returns the keyboard to the terminal.
func (a *App) closeFind() {
	if a.find.term != nil {
		a.find.term.EndSearch()
	}
	a.find = findState{query: a.find.query}
	a.focusTerm = nil
	a.mu.Lock()
	a.peek.focused = false
	a.mu.Unlock()
	a.Repaint()
}

// findBar floats inside the searched view, without resizing its PTY.
func (a *App) findBar(c *ui.Context, key string, term *terminal.Terminal) {
	f := &a.find
	if f.key != key || (a.isOverlay() && a.overlay != "peek") {
		return
	}
	if f.term != term {
		if f.term != nil {
			f.term.EndSearch()
		}
		f.term = term
	}
	if term.SearchStatus().Text != f.query {
		term.Search(f.query)
	}
	t := a.T
	bar := ui.Row(c.Key("terminal-find")).Absolute().Top(8).Right(8).
		Width(390).MaxWidthPercent(96).Height(36).PaddingX(8).Gap(5).
		AlignItems(ui.Center).Radius(8).Background(t.Popover).Border(1, t.Border).Label("Find in terminal")
	bar.Children(func() {
		a.icon(c, "search", 13, t.Dim)
		input := ui.TextInputBase(c.Key("query"), &f.query).Grow(1).MinWidth(0).
			FontSize(a.fontSize(13)).TextColor(t.Foreground).Placeholder("Find in terminal…").Label("Terminal search")
		if f.focus {
			f.focus = false
			a.focusTerm = term
			input.Focus().SetTextSelection(0, utf8.RuneCountInString(f.query))
		}
		if input.Changed() {
			term.Search(f.query)
		}
		if input.Submitted() {
			a.findStep(c.Modifiers()&ui.Shift != 0)
		}
		st := term.SearchStatus()
		count := ""
		switch {
		case f.query == "":
		case st.Total > 0:
			count = fmt.Sprintf("%d of %d", st.Selected, st.Total)
		case st.Searching:
			count = "Searching…"
		default:
			count = "No matches"
		}
		ui.Text(c, count).FontSize(a.fontSize(11)).TextColor(t.Dim).SingleLine().Shrink(0)
		a.findButton(c, "↑", "Next match (Enter)", term.SearchNext)
		a.findButton(c, "↓", "Previous match (Shift+Enter)", term.SearchPrevious)
		a.findButton(c, "✕", "Close terminal search (Escape)", a.closeFind)
	})
	if bar.Shortcut(0, ui.KeyEscape) {
		a.closeFind()
	}
}

func (a *App) findButton(c *ui.Context, text, label string, click func()) {
	b := ui.ButtonBase(c).Size(22, 24).Radius(4).Center().Cursor(ui.CursorPointer).Label(label).Tooltip(label)
	if b.Hovered() {
		b.Background(a.T.Hover)
	}
	b.Children(func() { ui.Text(c, text).FontSize(a.fontSize(13)).TextColor(a.T.Muted) })
	if b.Clicked() {
		click()
	}
}
