package ui

import "github.com/egoist/mygo/plugins/terminal"

// terminalPreferences is shared with attachment goroutines under App.mu.
type terminalPreferences struct {
	clipboard bool
	contrast  float32
}

// currentTerminalOptions gives panes and peek the same saved preferences.
func (a *App) currentTerminalOptions() terminal.Options {
	a.mu.Lock()
	defer a.mu.Unlock()
	return terminal.Options{
		Font:             a.terminalStyle,
		NoClipboardWrite: !a.terminalPrefs.clipboard,
		MinimumContrast:  a.terminalPrefs.contrast,
	}
}

// applyTerminalPreferences keeps text contrast current without replacing
// attachments, their output, selection or scroll position. Clipboard write
// access currently takes effect on new views: MyGo has no setter for it.
func (a *App) applyTerminalPreferences() {
	prefs := terminalPreferences{clipboard: a.settings.Clipboard, contrast: float32(a.settings.MinContrast)}
	a.mu.Lock()
	defer a.mu.Unlock()
	previous := a.terminalPrefs
	a.terminalPrefs = prefs
	if prefs.contrast == previous.contrast {
		return
	}
	for _, entry := range a.terms {
		entry.term.SetMinimumContrast(prefs.contrast)
	}
	if a.peek.term != nil {
		a.peek.term.SetMinimumContrast(prefs.contrast)
	}
}
