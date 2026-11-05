// Kero's tones and washes are adapted from github.com/egoist/kero/theme.go.
// Copyright (c) 2026 EGOIST. MIT license: ../../docs/kero-LICENSE.txt.
package ui

import (
	"math"

	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"
)

// windowFrame paints the wash and glow behind the window's content, as in Kero.
func (t *Theme) windowFrame(c *ui.Context) ui.Element {
	frame := ui.Row(c).Fill().Background(t.Canvas).Font(UIFont).AlignItems(ui.Start)
	if len(t.Backdrop.Stops) > 0 {
		c.Root().LinearGradient(t.Backdrop)
	}
	if len(t.Glow.Stops) > 0 {
		// Absolute children paint after the panes and would tint their text.
		frame.LinearGradient(t.Glow)
	}
	return frame
}

// keroThemes keeps Kero's lowercase names distinct from Ghostty's names.
var keroThemes = []string{"petal", "mist", "meadow", "dusk", "tide", "apricot", "aurora", "dune", "neutral", "plain"}

func themeForAppearance(name string, dark bool) *Theme {
	if w, ok := washTable[name]; ok {
		if name == "plain" {
			if dark {
				return resolve(graphite)
			}
			return resolve(paper)
		}
		term, accent := toneTheme(w.tone, dark)
		th := themeFromTerminal(term)
		th.Accent = accent
		colors, glow := w.light, w.lightGlow
		if dark {
			colors, glow = w.dark, w.darkGlow
		}
		stops := make([]ui.GradientStop, len(colors))
		for i, color := range colors {
			stops[i] = ui.GradientStop{Color: ui.Hex(color), At: float32(i) / float32(len(colors)-1)}
		}
		th.Backdrop = ui.LinearGradient{Angle: 180, Oklab: true, Stops: stops}
		th.Glow = ui.LinearGradient{Angle: 225, Oklab: true, Stops: []ui.GradientStop{{Color: glow, At: 0}, {Color: glow.Alpha(0), At: 0.5}}}
		th.Canvas, th.Chrome = ui.Color{}, Alpha(th.Background, 0.25)
		return th
	}
	switch name {
	case "graphite":
		return resolve(graphite)
	case "ink":
		return resolve(ink)
	case "moss":
		return resolve(moss)
	case "paper":
		return resolve(paper)
	}
	if term, err := terminal.GhosttyTheme(name); err == nil {
		return themeFromTerminal(term)
	}
	if dark {
		return resolve(graphite)
	}
	return resolve(paper)
}

// themeFromTerminal colors the whole interface from the terminal palette.
func themeFromTerminal(term *terminal.Theme) *Theme {
	p := palette{background: colorValue(term.Background), foreground: colorValue(term.Foreground), selection: colorValue(term.Selection)}
	for i, c := range term.Palette {
		p.ansi[i] = colorValue(c)
	}
	p.status = [3]uint32{p.ansi[3], p.ansi[1], p.ansi[4]}
	th := resolve(p)
	th.Foreground, th.TerminalForeground, th.Background = term.Foreground, term.Foreground, term.Background
	th.Cursor, th.Selection, th.Ansi = term.Cursor, term.Selection, term.Palette
	th.CursorText, th.SelectionText = term.CursorText, term.SelectionText
	th.Muted, th.Dim = term.Background.Mix(term.Foreground, 0.64), term.Background.Mix(term.Foreground, 0.42)
	th.Accent = term.Palette[4]
	copy := *term
	th.terminalColors = &copy
	shade := term.Background.Mix(ui.Hex("#000000"), 0.08)
	th.Backdrop = ui.LinearGradient{Angle: 180, Oklab: true, Stops: []ui.GradientStop{{Color: shade, At: 0}, {Color: term.Background, At: 1}}}
	th.Glow = ui.LinearGradient{Angle: 225, Oklab: true, Stops: []ui.GradientStop{{Color: Alpha(th.Accent, 0.16), At: 0}, {Color: Alpha(th.Accent, 0), At: 0.5}}}
	th.Canvas, th.Chrome = ui.Color{}, Alpha(th.Background, 0.25)
	return th
}

func colorValue(c ui.Color) uint32 { return uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B) }

func (s settingsState) themeFor(dark bool) (string, *Theme) {
	name := s.LightTheme
	if dark {
		name = s.DarkTheme
	}
	if name != "" {
		return name, themeForAppearance(name, dark)
	}
	appearance := AppearanceLight
	if dark {
		appearance = AppearanceDark
	}
	return ThemeFor(s.Theme, appearance, dark)
}

// chooseTheme changes just one appearance, leaving the other choice intact.
func (a *App) chooseTheme(name string, dark bool) {
	if !knownTheme(name) {
		return
	}
	if dark {
		a.settings.DarkTheme = name
	} else {
		a.settings.LightTheme = name
	}
	a.applyTheme()
}

// tone is the hue of a kero theme, how much of it its cards and text
// take, how vivid the programs' colors are, which lean to its hue, and
// the hue of its accent: the cursor, selection and controls.
type tone struct{ hue, tint, vivid, accent float32 }

// toneTheme is a kero theme's terminal colors, and its accent: its cards
// and text tinted with its hue, and the programs' colors muted to sit
// with them, the cursor and selection in its hue.
func toneTheme(t tone, dark bool) (*terminal.Theme, ui.Color) {
	tint := func(l, c float32) ui.Color { return ui.Oklch(l, c*t.tint, t.hue) }
	// Red, green, yellow, blue, magenta and cyan.
	hues := [6]float32{28, 145, 85, 255, 325, 195}
	ls := [6]float32{0.56, 0.56, 0.62, 0.54, 0.54, 0.56}
	cs := [6]float32{0.15, 0.13, 0.12, 0.13, 0.14, 0.09}
	bright := float32(0.05)
	th := &terminal.Theme{}
	var accent ui.Color
	if dark {
		ls = [6]float32{0.70, 0.74, 0.80, 0.70, 0.72, 0.76}
		cs = [6]float32{0.14, 0.13, 0.12, 0.12, 0.13, 0.10}
		bright = 0.06
		th.Background, th.Foreground = tint(0.255, 0.02), tint(0.92, 0.015)
		th.Palette[0], th.Palette[7], th.Palette[8], th.Palette[15] = tint(0.36, 0.02), tint(0.82, 0.012), tint(0.5, 0.02), tint(0.96, 0.01)
		th.Selection = ui.Oklch(0.38, 0.06, t.accent)
		accent = ui.Oklch(0.74, 0.13, t.accent)
	} else {
		// The less tinted, the whiter the paper: an untinted theme's
		// cards are white.
		paper := 0.978 + 0.022*max(0, 1-t.tint/0.5)
		th.Background, th.Foreground = tint(paper, 0.012), tint(0.28, 0.03)
		th.Palette[0], th.Palette[7], th.Palette[8], th.Palette[15] = th.Foreground, tint(0.7, 0.02), tint(0.52, 0.02), tint(0.8, 0.015)
		th.Selection = ui.Oklch(0.9, 0.045, t.accent)
		accent = ui.Oklch(0.55, 0.13, t.accent)
	}
	for i := range 6 {
		// Toward the theme's hue, the less the farther it is, as a far
		// color turned is a muddy one, and more vivid the nearer to it: a
		// theme's own color stands out of its palette.
		d := float32(math.Remainder(float64(t.hue-hues[i]), 360))
		h := hues[i] + d*0.35*(1-float32(math.Abs(float64(d)))/180)
		c := cs[i] * t.vivid * (1 + 0.45*max(0, 1-float32(math.Abs(float64(d)))/60))
		th.Palette[1+i] = ui.Oklch(ls[i], c, h)
		th.Palette[9+i] = ui.Oklch(ls[i]+bright, c+0.01, h)
	}
	th.Cursor, th.CursorText, th.SelectionText = accent, th.Background, th.Foreground
	return th, accent
}

type wash struct {
	light, dark         []string
	lightGlow, darkGlow ui.Color
	tone                tone
}

var washTable = map[string]wash{
	// Rex's: pink lavender above, rose at the top right, butter cream below.
	"petal": {
		light: []string{"#ebe0f3", "#f3e7f1", "#f7eee4", "#f6efcf"}, lightGlow: ui.RGBA(247, 220, 241, 0.75),
		dark: []string{"#240f1d", "#26111a", "#271610"}, darkGlow: ui.RGBA(142, 58, 110, 0.3),
		tone: tone{350, 1, 0.95, 350},
	},
	"mist": {
		light: []string{"#e3e9f4", "#ebeff6", "#f1f2f3", "#f2f1ec"}, lightGlow: ui.RGBA(210, 225, 248, 0.75),
		dark: []string{"#0c1628", "#0d1827", "#111b23"}, darkGlow: ui.RGBA(46, 91, 166, 0.3),
		tone: tone{255, 1, 0.8, 255},
	},
	"meadow": {
		light: []string{"#e2eee5", "#ebf2e7", "#f3f2e2", "#f5f0d6"}, lightGlow: ui.RGBA(214, 240, 222, 0.75),
		dark: []string{"#081c0d", "#101d0c", "#1b1c0b"}, darkGlow: ui.RGBA(6, 113, 50, 0.3),
		tone: tone{140, 1, 0.9, 140},
	},
	"dusk": {
		light: []string{"#e4dbf3", "#efdeee", "#f6e3dc", "#f7e6cf"}, lightGlow: ui.RGBA(246, 206, 222, 0.8),
		dark: []string{"#190f2e", "#260f24", "#2d140b"}, darkGlow: ui.RGBA(125, 65, 138, 0.3),
		tone: tone{315, 1, 1.1, 315},
	},
	// Sea glass: aqua above, a pale tide line below.
	"tide": {
		light: []string{"#dbebef", "#e3eff1", "#ecf3f0", "#f2f2e8"}, lightGlow: ui.RGBA(194, 232, 238, 0.75),
		dark: []string{"#001b22", "#001e1f", "#0b1e1b"}, darkGlow: ui.RGBA(0, 112, 128, 0.3),
		tone: tone{205, 1, 0.85, 205},
	},
	"apricot": {
		light: []string{"#f5e1d6", "#f8e7dc", "#f8eedf", "#f6f0da"}, lightGlow: ui.RGBA(252, 212, 194, 0.78),
		dark: []string{"#2a1207", "#291507", "#25190a"}, darkGlow: ui.RGBA(151, 62, 22, 0.3),
		tone: tone{50, 1, 1.05, 50},
	},
	// Mint at the top, fading through blue to violet.
	"aurora": {
		light: []string{"#dbefe8", "#e1ebf2", "#e8e4f4", "#f0e6f1"}, lightGlow: ui.RGBA(198, 238, 224, 0.75),
		dark: []string{"#001f13", "#06192b", "#1e122a"}, darkGlow: ui.RGBA(0, 115, 75, 0.3),
		tone: tone{170, 1, 1.15, 170},
	},
	"dune": {
		light: []string{"#ede4d8", "#f1e9de", "#f4eee4", "#f3efe8"}, lightGlow: ui.RGBA(242, 222, 194, 0.7),
		dark: []string{"#22190b", "#211a0e", "#201b12"}, darkGlow: ui.RGBA(135, 79, 0, 0.3),
		tone: tone{80, 0.7, 0.55, 80},
	},
	// White paper on a desk of light gray, in ink, flat and quiet.
	"neutral": {
		light: []string{"#f6f6f7", "#f7f7f8", "#f8f8f8"}, lightGlow: ui.RGBA(255, 255, 255, 0),
		dark: []string{"#141416", "#151517", "#161618"}, darkGlow: ui.RGBA(90, 120, 200, 0.1),
		tone: tone{260, 0, 0.9, 262},
	},
	"plain": {
		light: []string{"#eceef4", "#eceef4"}, lightGlow: ui.RGBA(0, 0, 0, 0),
		dark: []string{"#141418", "#141418"}, darkGlow: ui.RGBA(0, 0, 0, 0),
	},
}
