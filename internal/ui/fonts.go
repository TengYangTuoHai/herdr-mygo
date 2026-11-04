package ui

import (
	"io/fs"
	"log"

	"github.com/egoist/mygo/ui"

	"herdr-mygo/internal/assets"
)

// RegisterFonts adds JetBrains Mono with Nerd Fonts' prompt and TUI icons
// for terminals, and Geist for the panels. Keep the terminal family name so
// existing settings and terminal views use the bundled glyphs.
func RegisterFonts() {
	for _, font := range []struct{ family, name string }{
		{TerminalFont, "fonts/JetBrainsMonoNerdFont-Regular.ttf"},
		{TerminalFont, "fonts/JetBrainsMonoNerdFont-Bold.ttf"},
		{TerminalFont, "fonts/JetBrainsMonoNerdFont-Italic.ttf"},
		{TerminalFont, "fonts/JetBrainsMonoNerdFont-BoldItalic.ttf"},
		{UIFont, "fonts/Geist.ttf"},
	} {
		data, err := fs.ReadFile(assets.Files, font.name)
		if err != nil {
			log.Printf("font %s: %v", font.name, err)
			continue
		}
		if err := ui.RegisterFont(data, font.family); err != nil {
			log.Printf("font %s: %v", font.name, err)
		}
	}
}
