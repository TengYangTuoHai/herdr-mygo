package ui

import (
	"io/fs"
	"log"

	"github.com/egoist/mygo/ui"

	"herdr-mygo/internal/assets"
)

// RegisterFonts adds the app's own fonts: JetBrains Mono for terminals, Geist
// for the panels.
func RegisterFonts() {
	for family, name := range map[string]string{
		"JetBrains Mono": "fonts/JetBrainsMono-Variable.ttf",
		"Geist":          "fonts/Geist.ttf",
	} {
		data, err := fs.ReadFile(assets.Files, name)
		if err != nil {
			continue
		}
		if err := ui.RegisterFont(data, family); err != nil {
			log.Printf("font %s: %v", family, err)
		}
	}
}
