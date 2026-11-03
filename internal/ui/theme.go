package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Theme is every color the window draws with, resolved from a terminal
// palette the way Herdr derives its chrome (settings.rs, resolve).
type Theme struct {
	Background, TerminalForeground, Foreground      ui.Color
	Surface, Canvas, Chrome, Popover, Raised        ui.Color
	Hover, Selected, Border, Input                  ui.Color
	Muted, Dim, Accent, AccentForeground            ui.Color
	Solid, SolidForeground, Ring, Selection, Cursor ui.Color
	Working, Blocked, Done, Idle, Unknown           ui.Color
	Ansi                                            [16]ui.Color
	Dark                                            bool
}

type palette struct {
	background, foreground, selection, cursor uint32
	status                                    [3]uint32
	ansi                                      [16]uint32
}

var graphite = palette{
	background: 0x17191d,
	foreground: 0xd2d4d9,
	selection:  0x394454,
	status:     [3]uint32{0xe5c07b, 0xf0705f, 0x76a8ff},
	ansi: [16]uint32{
		0x45484f, 0xdc8c95, 0xa6bd91, 0xd8be87, 0x96b2d7, 0xb5a2ce, 0x8cbdc3, 0xd2d4d9,
		0x818793, 0xf0a5ad, 0xbad5a2, 0xead2a1, 0xb0cbef, 0xcbb7e6, 0xa4d5db, 0xf0f0f2,
	},
}

var ink = palette{
	background: 0x111720,
	foreground: 0xcdd6e4,
	selection:  0x344861,
	status:     [3]uint32{0xe5c07b, 0xf0705f, 0x76a8ff},
	ansi: [16]uint32{
		0x414b5a, 0xe58c9a, 0x89c5aa, 0xd9be82, 0x91b6ee, 0xb9a0e2, 0x85c4d0, 0xcdd6e4,
		0x7f91a7, 0xf5a5b2, 0xa5dec3, 0xedd39c, 0xaccbfa, 0xd0b7f4, 0xa0dce5, 0xf0f5fc,
	},
}

var moss = palette{
	background: 0x151c19,
	foreground: 0xcdd7ce,
	selection:  0x354a40,
	status:     [3]uint32{0xe5c07b, 0xf0705f, 0x76a8ff},
	ansi: [16]uint32{
		0x435248, 0xdf978d, 0xa0c4a2, 0xd4bc8b, 0x99b9cd, 0xbba4c8, 0x8bbdb5, 0xcdd7ce,
		0x82968a, 0xf1afa3, 0xb8dab9, 0xe9d2a5, 0xb1d0e2, 0xd1bddc, 0xa4d6ca, 0xeff5ee,
	},
}

var paper = palette{
	background: 0xffffff,
	foreground: 0x25272c,
	selection:  0xcfdcf5,
	cursor:     0x25272c,
	status:     [3]uint32{0x9a6700, 0xc93c2c, 0x2f6fe4},
	ansi: [16]uint32{
		0x25272c, 0xc62a2a, 0x1a7f37, 0x8f5d00, 0x1f62d0, 0x7a45c8, 0x0f6f74, 0x5f6068,
		0x7b7e87, 0xd1242f, 0x1f883d, 0x9a6700, 0x0969da, 0x8250df, 0x1b7c83, 0x111214,
	},
}

// themes are the themes the settings offer, in the order they show.
var themes = []string{"graphite", "ink", "moss", "paper"}

// knownTheme reports whether name is one of them.
func knownTheme(name string) bool {
	for _, t := range themes {
		if t == name {
			return true
		}
	}
	return false
}

// ThemeByName resolves a built-in theme, graphite when the name is unknown.
func ThemeByName(name string) *Theme {
	var p palette
	switch name {
	case "ink":
		p = ink
	case "moss":
		p = moss
	case "paper":
		p = paper
	default:
		p = graphite
	}
	return resolve(p)
}

// Appearance is how the window picks light or dark: from the desktop, or
// always one of them.
const (
	AppearanceSystem = iota
	AppearanceLight
	AppearanceDark
)

// ThemeFor resolves the theme the window draws with, and its name: the one
// picked when its own light or dark fits the appearance, else paper in the
// light and graphite in the dark. systemDark is the desktop's appearance,
// read where the appearance asks for the system's.
func ThemeFor(name string, appearance int, systemDark bool) (string, *Theme) {
	dark := systemDark
	switch appearance {
	case AppearanceLight:
		dark = false
	case AppearanceDark:
		dark = true
	}
	if picked := ThemeByName(name); picked.Dark == dark {
		return name, picked
	}
	if dark {
		return "graphite", ThemeByName("graphite")
	}
	return "paper", ThemeByName("paper")
}

func isLight(c uint32) bool {
	ch := func(s uint) float64 { return float64((c>>s)&255) / 255 }
	return 0.2126*ch(16)+0.7152*ch(8)+0.0722*ch(0) > 0.5
}

func mix(a, b uint32, t float64) uint32 {
	var out uint32
	for _, s := range []uint{16, 8, 0} {
		x := float64((a >> s) & 255)
		y := float64((b >> s) & 255)
		out |= uint32(math.Round(x*(1-t)+y*t)) << s
	}
	return out
}

func rgb(c uint32) ui.Color {
	return ui.RGB(uint8((c>>16)&255), uint8((c>>8)&255), uint8(c&255))
}

func rgba(hex string) ui.Color { return ui.Hex(hex) }

func hex(c uint32) string { return fmt.Sprintf("#%06x", c) }

func resolve(p palette) *Theme {
	bg := p.background
	light := isLight(bg)
	fg := uint32(0xeeeeef)
	if light {
		fg = 0x1b1b1e
	}
	var popover, raised, hover, selected, border, input, canvas, chrome string
	if light {
		warm := uint32(0xe8e8e2)
		popover = hex(bg)
		raised = hex(mix(bg, 0, 0.035))
		hover = "#0000000c"
		selected = "#00000012"
		border = "#00000017"
		input = "#00000026"
		canvas = hex(mix(bg, warm, 0.2))
		chrome = hex(mix(bg, warm, 0.6))
	} else {
		popover = hex(mix(bg, 0xffffff, 0.06))
		raised = hex(mix(bg, 0xffffff, 0.09))
		hover = "#ffffff0e"
		selected = "#ffffff17"
		border = "#ffffff18"
		input = "#ffffff24"
		canvas = hex(mix(bg, 0, 0.16))
		chrome = hex(mix(bg, 0, 0.32))
	}
	mutedT := 0.6
	dimT := 0.38
	if light {
		mutedT = 0.66
		dimT = 0.46
	}
	solidFg := mix(bg, 0, 0.28)
	if light {
		solidFg = 0xffffff
	}
	t := &Theme{
		Background:         rgb(bg),
		TerminalForeground: rgb(p.foreground),
		Foreground:         rgb(fg),
		Surface:            rgb(bg),
		Canvas:             rgba(canvas),
		Chrome:             rgba(chrome),
		Popover:            rgba(popover),
		Raised:             rgba(raised),
		Hover:              rgba(hover),
		Selected:           rgba(selected),
		Border:             rgba(border),
		Input:              rgba(input),
		Muted:              rgb(mix(bg, fg, mutedT)),
		Dim:                rgb(mix(bg, fg, dimT)),
		Accent:             rgb(p.ansi[4]),
		AccentForeground:   rgb(bg),
		Solid:              rgb(fg),
		SolidForeground:    rgb(solidFg),
		Ring:               rgb(mix(bg, fg, 0.7)),
		Selection:          rgb(p.selection),
		Cursor:             rgb(p.foreground),
		Working:            rgb(p.status[0]),
		Blocked:            rgb(p.status[1]),
		Done:               rgb(p.status[2]),
		Idle:               rgb(mix(bg, fg, 0.38)),
		Unknown:            rgb(mix(bg, fg, 0.38)),
		Dark:               !light,
	}
	if p.cursor != 0 {
		t.Cursor = rgb(p.cursor)
	}
	for i, c := range p.ansi {
		t.Ansi[i] = rgb(c)
	}
	return t
}

// Status returns the color of an agent status word.
func (t *Theme) Status(word string) ui.Color {
	switch word {
	case "working":
		return t.Working
	case "blocked":
		return t.Blocked
	case "done":
		return t.Done
	case "idle":
		return t.Idle
	}
	return t.Dim
}

// Alpha returns c at the given opacity, as Herdr writes "#rrggbb26".
func Alpha(c ui.Color, a float32) ui.Color {
	return ui.RGBA(c.R, c.G, c.B, a)
}

// named returns a theme color by the name mods use, or a literal #rrggbb.
func (t *Theme) named(name string) ui.Color {
	if strings.HasPrefix(name, "#") {
		return ui.Hex(name)
	}
	switch name {
	case "background":
		return t.Background
	case "surface":
		return t.Surface
	case "canvas":
		return t.Canvas
	case "chrome":
		return t.Chrome
	case "popover":
		return t.Popover
	case "raised":
		return t.Raised
	case "hover":
		return t.Hover
	case "selected":
		return t.Selected
	case "border":
		return t.Border
	case "input":
		return t.Input
	case "foreground":
		return t.Foreground
	case "muted":
		return t.Muted
	case "dim":
		return t.Dim
	case "accent":
		return t.Accent
	case "accent_foreground":
		return t.AccentForeground
	case "solid":
		return t.Solid
	case "solid_foreground":
		return t.SolidForeground
	case "ring":
		return t.Ring
	case "selection":
		return t.Selection
	case "cursor":
		return t.Cursor
	case "working":
		return t.Working
	case "blocked":
		return t.Blocked
	case "done":
		return t.Done
	case "idle":
		return t.Idle
	case "unknown":
		return t.Unknown
	}
	return ui.Hex(name)
}
