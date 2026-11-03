package ui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"herdr-mygo/internal/herdr"
)

// settingsEntry is one searchable setting. The search and the sections both
// use the same names, so a result and its row say the same thing.
type settingsEntry struct {
	Section     string
	SectionName string
	Title       string
	About       string
	Value       string
	Icon        string
}

var (
	appearanceNames    = []string{"System", "Light", "Dark"}
	optionAltNames     = []string{"Auto", "Left", "Right", "Off"}
	updateChannelNames = []string{"Stable", "Preview"}
)

// settingsSectionNames labels every section, in the nav's order.
var settingsSectionNames = map[string]string{
	"look": "Look", "terminal": "Terminal", "machines": "Machines",
	"alerts": "Alerts", "updates": "Updates", "privacy": "Privacy",
}

// settingsSectionIcons is the icon each section's rows and results carry.
var settingsSectionIcons = map[string]string{
	"look": "sun", "terminal": "terminal", "machines": "server",
	"alerts": "bell", "updates": "refresh-cw", "privacy": "shield",
}

// settingKeywords are what a person might type to find a setting whose name
// does not contain the word itself.
var settingKeywords = map[string]string{
	"Theme":                     "appearance accent colors look",
	"Appearance":                "theme light dark system mode",
	"Accent":                    "theme color colors highlight",
	"Interface font":            "font typeface size type",
	"Terminal font":             "font typeface mono terminal",
	"Zoom":                      "size scale bigger smaller",
	"Option as Alt":             "option alt macos keyboard",
	"Clipboard":                 "copy paste",
	"Minimum contrast":          "contrast readability",
	"Machines":                  "ssh host server remote",
	"Keep SSH connections warm": "ssh keepalive connection",
	"Done":                      "alert sound toast finish complete",
	"Needs you":                 "alert sound toast approval question",
	"Play a sound":              "sound audio alert",
	"Done sound":                "sound audio alert",
	"Needs you sound":           "sound audio alert",
	"By agent":                  "sound agent override",
	"Channel":                   "update stable preview",
	"Install automatically":     "update automatic download",
	"Check in the background":   "update check background",
	"herdr-core":                "core hc version",
	"Anonymous usage counts":    "telemetry privacy analytics",
	"Reports":                   "telemetry privacy report",
}

func onOff(on bool) string {
	if on {
		return "On"
	}
	return "Off"
}

func pick(names []string, at int) string {
	if at < 0 || at >= len(names) {
		return names[0]
	}
	return names[at]
}

// settingsEntries is every setting, with the value it shows now.
func (a *App) settingsEntries() []settingsEntry {
	s := a.settings
	zoom := a.Zoom
	if zoom <= 0 {
		zoom = 1
	}
	entry := func(section, title, about, value string) settingsEntry {
		return settingsEntry{
			Section: section, SectionName: settingsSectionNames[section],
			Title: title, About: about, Value: value,
			Icon: settingsSectionIcons[section],
		}
	}
	return []settingsEntry{
		entry("look", "Appearance", "System, or always light or dark.", pick(appearanceNames, s.Appearance)),
		entry("look", "Theme", "Colors for the chrome and every terminal.", strings.Title(a.currentThemeName())),
		entry("look", "Accent", "Focus rings, selections and key hints.", strings.ToUpper(s.Accent)),
		entry("look", "Interface font", "The panels’ typeface and size.", fmt.Sprintf("%s %d", s.InterfaceFont, s.InterfaceSize)),
		entry("look", "Terminal font", "Used by every pane, on every machine.", fmt.Sprintf("%s %d", s.TerminalFont, s.TerminalSize)),
		entry("look", "Zoom", "Scales the whole window, terminals included.", fmt.Sprintf("%.0f%%", zoom*100)),

		entry("terminal", "Option as Alt", "Send Option as Alt, or keep it for macOS accents.", pick(optionAltNames, s.OptionAsAlt)),
		entry("terminal", "Clipboard", "Let terminal programs write the clipboard.", onOff(s.Clipboard)),
		entry("terminal", "Minimum contrast", "Raise terminal text contrast. 1.0 leaves program colors alone.", fmt.Sprintf("%.1f", s.MinContrast)),

		entry("machines", "Machines", "Add, connect and remove the computers Herdr reaches over ssh.", "Manage"),
		entry("machines", "Keep SSH connections warm", "Adds keepalives and reuses connections.", onOff(s.KeepSSHWarm)),

		entry("alerts", "Done", "A sound and a toast when an agent finishes while you are elsewhere.", onOff(s.AlertDone)),
		entry("alerts", "Needs you", "When an agent asks a question or waits for approval.", onOff(s.AlertNeeds)),
		entry("alerts", "Play a sound", "Use the built-in sounds, or choose your own files.", onOff(s.AlertSound)),
		entry("alerts", "Done sound", "Plays when an agent finishes.", "Glass"),
		entry("alerts", "Needs you sound", "Plays when an agent asks something.", "Ping"),
		entry("alerts", "By agent", "Turn a sound off for one agent without changing the rest.", "All agents on"),

		entry("updates", "Channel", "Stable, or preview releases early.", pick(updateChannelNames, s.UpdateChannel)),
		entry("updates", "Install automatically", "Download and install when Herdr is not running.", onOff(s.AutoInstall)),
		entry("updates", "Check in the background", "Once a day, without interrupting a terminal.", onOff(s.BackgroundCheck)),
		entry("updates", "herdr-core", "The background program that runs your terminals.", "Connected"),

		entry("privacy", "Anonymous usage counts", "Daily counts only: versions, machines, agents and commands.", onOff(s.Telemetry)),
		entry("privacy", "Reports", "At most one a day. Nothing comes from herdr-core.", "Daily"),
	}
}

// settingsResults is the entries matching the search, best first.
func (a *App) settingsResults() []settingsEntry {
	all := a.settingsEntries()
	q := strings.ToLower(strings.TrimSpace(a.settings.Search))
	if q == "" {
		return all
	}
	q = strings.ReplaceAll(q, " ", "")
	type scored struct {
		entry settingsEntry
		score int
	}
	var hits []scored
	for _, e := range all {
		if score, ok := fuzzyScore(q, strings.ToLower(e.Title)); ok {
			hits = append(hits, scored{e, score + 10})
			continue
		}
		haystack := strings.ToLower(e.SectionName + " " + settingKeywords[e.Title])
		if score, ok := fuzzyScore(q, haystack); ok {
			hits = append(hits, scored{e, score})
		}
	}
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && hits[j].score > hits[j-1].score; j-- {
			hits[j], hits[j-1] = hits[j-1], hits[j]
		}
	}
	out := make([]settingsEntry, len(hits))
	for i, h := range hits {
		out[i] = h.entry
	}
	return out
}

// settingsMove keeps the search selection on a result.
func (a *App) settingsMove(delta int) {
	n := len(a.settingsResults())
	if n == 0 {
		a.settings.SearchAt = 0
		return
	}
	at := a.settings.SearchAt + delta
	if at < 0 {
		at = 0
	}
	if at >= n {
		at = n - 1
	}
	a.settings.SearchAt = at
}

// chooseSettingsResult opens the section the chosen result lives in.
func (a *App) chooseSettingsResult(i int) {
	results := a.settingsResults()
	if i < 0 || i >= len(results) {
		return
	}
	a.settings.Section = results[i].Section
	a.settings.SearchOpen = false
	a.settings.Search = ""
	a.settings.SearchAt = 0
	a.Repaint()
}

// openSettingsSearch starts a search with an empty field.
func (a *App) openSettingsSearch() {
	if a.settings.SearchOpen {
		return
	}
	a.settings.SearchOpen = true
	a.settings.Search = ""
	a.settings.SearchAt = 0
	a.Repaint()
}

// closeSettingsSearch returns to the section alone.
func (a *App) closeSettingsSearch() {
	a.settings.SearchOpen = false
	a.settings.Search = ""
	a.settings.SearchAt = 0
	a.Repaint()
}

const (
	settingsVersion   = "0.1.0"
	settingsWidth     = float32(1080)
	settingsHeight    = float32(720)
	settingsNavWidth  = float32(224)
	settingsCardAlpha = float32(0.035)
)

func minFloat(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// settingsOverlay is the Settings screen: a header, a section nav and the
// chosen section's controls.
func (a *App) settingsOverlay(c *ui.Context) {
	t := a.T
	if a.settings.Section == "" {
		a.settings.Section = "look"
	}
	width := minFloat(settingsWidth, a.W-80)
	height := minFloat(settingsHeight, a.H-40)
	panel := a.card(c, width, height).Absolute().
		Top((a.H - height) / 2).Left((a.W - width) / 2)
	panel.Children(func() {
		a.settingsHeader(c)
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		ui.Row(c).Grow(1).FillWidth().AlignItems(ui.Start).Children(func() {
			a.settingsNav(c)
			ui.Box(c).Width(1).FillHeight().Background(t.Border)
			a.settingsContent(c)
		})
	})
}

// settingsHeader is the title, the search field and the close button.
func (a *App) settingsHeader(c *ui.Context) {
	t := a.T
	ui.Row(c).Height(58).FillWidth().Padding(0, 16, 0, 22).Gap(12).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Settings").FontSize(15.5).FontWeight(600).TextColor(t.Foreground)
		ui.Spacer(c)
		a.settingsSearchField(c)
		close := ui.Box(c).Size(28, 28).Radius(7).Center()
		if close.Hovered() {
			close.Background(t.Hover)
		}
		close.Cursor(ui.CursorPointer)
		close.Children(func() { a.icon(c, "x", 14, t.Dim) })
		if close.Clicked() {
			a.CloseOverlay()
		}
	})
}

// settingsSearchField is the field in the header, or a button that opens it.
func (a *App) settingsSearchField(c *ui.Context) {
	t := a.T
	field := ui.Row(c).Width(230).Height(30).PaddingX(9).Gap(8).Radius(8).
		Border(1, t.Border).Background(Alpha(t.Foreground, 0.04)).
		AlignItems(ui.Center)
	if a.settings.SearchOpen {
		field.Border(1, t.Accent).Background(t.Background)
		field.Children(func() {
			a.icon(c, "search", 14, t.Dim)
			input := ui.TextInputBase(c, &a.settings.Search).Grow(1).FontSize(12.5).
				TextColor(t.Foreground).Placeholder("Search settings").AutoFocus()
			input.HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind != ui.InputKeyDown {
					return false
				}
				switch ev.Key {
				case ui.KeyDown:
					a.settingsMove(1)
					a.Repaint()
					return true
				case ui.KeyUp:
					a.settingsMove(-1)
					a.Repaint()
					return true
				}
				return false
			})
			if input.Changed() {
				a.settings.SearchAt = 0
			}
			if input.Submitted() {
				a.chooseSettingsResult(a.settings.SearchAt)
			}
			a.settingsKeycap(c, "esc")
		})
		return
	}
	if field.Hovered() {
		field.Background(t.Hover)
	}
	field.Cursor(ui.CursorPointer)
	field.Children(func() {
		a.icon(c, "search", 14, t.Dim)
		ui.Text(c, "Search settings").FontSize(12.5).TextColor(t.Dim).Grow(1)
		a.settingsKeycap(c, "⌘F")
	})
	if field.Clicked() {
		a.openSettingsSearch()
	}
}

// settingsKeycap is a small key hint.
func (a *App) settingsKeycap(c *ui.Context, label string) {
	t := a.T
	ui.Box(c).Height(16).PaddingX(4).Radius(4).Border(1, t.Border).
		Background(Alpha(t.Foreground, 0.04)).Center().Children(func() {
		ui.Text(c, label).FontSize(10).TextColor(t.Dim)
	})
}

// settingsNav is the sections list.
func (a *App) settingsNav(c *ui.Context) {
	t := a.T
	nav := ui.Column(c).Width(settingsNavWidth).FillHeight().Background(t.Chrome).
		Padding(8, 10, 12, 10)
	nav.Children(func() {
		navGroup := func(label string) {
			ui.Text(c, strings.ToUpper(label)).FontSize(10.5).FontWeight(600).
				LetterSpacing(0.6).TextColor(t.Dim).Padding(14, 10, 6, 10)
		}
		navRow := func(id, label, icon string) {
			on := a.settings.Section == id
			row := ui.Row(c).Height(34).FillWidth().PaddingX(10).Gap(10).
				Radius(8).AlignItems(ui.Center)
			if on {
				row.Background(t.Selected)
			} else if row.Hovered() {
				row.Background(t.Hover)
			}
			row.Cursor(ui.CursorPointer)
			row.Children(func() {
				color, iconColor := t.Muted, t.Dim
				if on {
					color, iconColor = t.Foreground, t.Accent
				}
				a.icon(c, icon, 15, iconColor)
				ui.Text(c, label).FontSize(13.5).TextColor(color)
			})
			if row.Clicked() {
				a.settings.Section = id
				a.settings.SearchOpen = false
				a.settings.Search = ""
				a.Repaint()
			}
		}

		navGroup("App")
		navRow("look", "Look", "sun")
		navRow("terminal", "Terminal", "terminal")
		navGroup("Herdr")
		navRow("machines", "Machines", "server")
		navRow("alerts", "Alerts", "bell")
		navRow("updates", "Updates", "refresh-cw")
		navRow("privacy", "Privacy", "shield")

		ui.Spacer(c)
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		ui.Row(c).Height(42).FillWidth().PaddingX(8).Gap(8).AlignItems(ui.Center).Children(func() {
			a.logo(c, 18)
			ui.Text(c, "Herdr").FontSize(11.5).TextColor(t.Muted)
			ui.Spacer(c)
			ui.Text(c, settingsVersion).Font(TerminalFont).FontSize(10.5).TextColor(t.Dim)
		})
	})
}

// settingsContent is the scrolling section, with search over it when open.
func (a *App) settingsContent(c *ui.Context) {
	area := ui.Column(c).Grow(1).FillHeight()
	area.Children(func() {
		ui.Scroll(c).Grow(1).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Padding(26, 32).Gap(14).Children(func() {
				a.settingsSection(c)
			})
		})
		if a.settings.SearchOpen {
			shade := ui.Box(c).Absolute().Fill()
			if a.T.Dark {
				shade.Background(ui.RGBA(18, 19, 22, 0.58))
			} else {
				shade.Background(ui.RGBA(245, 245, 241, 0.68))
			}
			a.settingsSearchResults(c)
		}
	})
}

// settingsSearchResults is the results card under the search field.
func (a *App) settingsSearchResults(c *ui.Context) {
	t := a.T
	results := a.settingsResults()
	if a.settings.SearchAt >= len(results) {
		a.settings.SearchAt = maxInt(0, len(results)-1)
	}
	shadow := ui.RGBA(0, 0, 0, 0.55)
	if !t.Dark {
		shadow = ui.RGBA(0, 0, 0, 0.18)
	}
	pop := ui.Column(c).Absolute().Top(10).Right(56).Width(404).Radius(12).
		Background(t.Popover).Border(1, t.Input).Shadow(0, 18, 50, 0, shadow).Clip()
	pop.Children(func() {
		ui.Row(c).FillWidth().Padding(10, 12, 7, 12).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("%d RESULTS", len(results))).FontSize(10).
				FontWeight(600).LetterSpacing(0.5).TextColor(t.Dim)
			ui.Spacer(c)
			if a.settings.Search != "" {
				ui.Text(c, a.settings.Search).FontSize(10).TextColor(t.Dim)
			}
		})
		if len(results) == 0 {
			ui.Text(c, fmt.Sprintf("Nothing matches %q.", a.settings.Search)).
				FontSize(12.5).TextColor(t.Dim).Padding(14, 12)
		} else {
			ui.Scroll(c).MaxHeight(288).FillWidth().Children(func() {
				ui.Column(c).FillWidth().Children(func() {
					for i, e := range results {
						row := ui.Row(c).Height(36).FillWidth().PaddingX(12).Gap(10).
							AlignItems(ui.Center)
						if i == a.settings.SearchAt {
							row.Background(t.Selected)
							row.ScrollIntoView()
						} else if row.Hovered() {
							row.Background(t.Hover)
						}
						row.Cursor(ui.CursorPointer)
						row.Children(func() {
							color, title := t.Dim, t.Muted
							if i == a.settings.SearchAt {
								color, title = t.Accent, t.Foreground
							}
							a.icon(c, e.Icon, 14, color)
							ui.Text(c, e.Title).FontSize(12.5).TextColor(title).Grow(1).
								Ellipsis("…")
							ui.Text(c, e.SectionName).FontSize(10.5).TextColor(t.Dim)
						})
						if row.Clicked() {
							a.chooseSettingsResult(i)
						}
					}
				})
			})
		}
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		ui.Row(c).FillWidth().Padding(7, 12).Gap(7).AlignItems(ui.Center).Children(func() {
			a.settingsKeycap(c, "↑↓")
			ui.Text(c, "move").FontSize(10.5).TextColor(t.Dim)
			a.settingsKeycap(c, "↵")
			ui.Text(c, "open").FontSize(10.5).TextColor(t.Dim)
			ui.Spacer(c)
			a.settingsKeycap(c, "esc")
			ui.Text(c, "close").FontSize(10.5).TextColor(t.Dim)
		})
	})
}

// settingsSection draws the chosen section.
func (a *App) settingsSection(c *ui.Context) {
	switch a.settings.Section {
	case "terminal":
		a.settingsTerminal(c)
	case "machines":
		a.settingsMachines(c)
	case "alerts":
		a.settingsAlerts(c)
	case "updates":
		a.settingsUpdates(c)
	case "privacy":
		a.settingsPrivacy(c)
	default:
		a.settingsLook(c)
	}
}

// settingsHead is a section's title and one line about it.
func (a *App) settingsHead(c *ui.Context, title, about string) {
	t := a.T
	ui.Column(c).Gap(4).Padding(0, 0, 4, 0).Children(func() {
		ui.Text(c, title).FontSize(21).FontWeight(600).TextColor(t.Foreground)
		ui.Text(c, about).FontSize(12.5).TextColor(t.Muted).Wrap()
	})
}

// settingsCard is one group of rows.
func (a *App) settingsCard(c *ui.Context, rows func()) {
	t := a.T
	card := ui.Column(c).FillWidth().Radius(12).Border(1, t.Border).
		Background(Alpha(t.Foreground, settingsCardAlpha)).Clip()
	card.Children(rows)
}

// settingsDivider separates rows inside a card.
func (a *App) settingsDivider(c *ui.Context) {
	ui.Box(c).Height(1).FillWidth().MarginX(16).Background(a.T.Border)
}

// settingsRow is a label, a line about it, and its control.
func (a *App) settingsRow(c *ui.Context, label, about string, control func()) {
	a.settingsRowWhen(c, true, label, about, control)
}

// settingsRowWhen draws a row that fades when its setting is off.
func (a *App) settingsRowWhen(c *ui.Context, on bool, label, about string, control func()) {
	t := a.T
	labelColor := t.Foreground
	if !on {
		labelColor = t.Dim
	}
	ui.Row(c).FillWidth().MinHeight(58).Padding(11, 16).Gap(24).
		AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, label).FontSize(13.5).FontWeight(500).TextColor(labelColor)
			if about != "" {
				ui.Text(c, about).FontSize(11.5).TextColor(t.Dim).Wrap()
			}
		})
		ctl := ui.Row(c).Gap(10).AlignItems(ui.Center).Shrink(0)
		if !on {
			ctl.Opacity(0.45)
		}
		ctl.Children(control)
	})
}

// settingsNote is the small line under a section's cards.
func (a *App) settingsNote(c *ui.Context, icon, text string) {
	t := a.T
	ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Padding(0, 2).Children(func() {
		a.icon(c, icon, 12, t.Dim)
		ui.Text(c, text).FontSize(11).TextColor(t.Dim).Wrap().Grow(1)
	})
}

// settingsToggle is an on/off switch.
func (a *App) settingsToggle(c *ui.Context, on *bool) {
	t := a.T
	track := ui.Box(c).Width(36).Height(21).Radius(11).Cursor(ui.CursorPointer)
	if *on {
		track.Background(t.Accent)
	} else {
		track.Background(Alpha(t.Foreground, 0.17))
	}
	track.Children(func() {
		ui.Row(c).Fill().Padding(2).AlignItems(ui.Center).Children(func() {
			if *on {
				ui.Spacer(c)
			}
			knob := ui.Box(c).Size(17, 17).Radius(9)
			if *on {
				knob.Background(t.AccentForeground)
			} else {
				knob.Background(t.Foreground)
			}
			if !*on {
				ui.Spacer(c)
			}
		})
	})
	if track.Clicked() {
		*on = !*on
		a.Repaint()
	}
}

// settingsSegmented is a row of mutually exclusive labels.
func (a *App) settingsSegmented(c *ui.Context, labels []string, value *int) {
	a.settingsSegmentedOn(c, labels, value, nil)
}

// settingsSegmentedOn is settingsSegmented that runs onChange with the
// index picked, for a setting that applies at once.
func (a *App) settingsSegmentedOn(c *ui.Context, labels []string, value *int, onChange func(int)) {
	t := a.T
	ui.Row(c).Padding(2).Radius(8).Gap(2).Background(Alpha(t.Foreground, 0.055)).
		AlignItems(ui.Center).Children(func() {
		for i, label := range labels {
			item := ui.Box(c).Height(26).PaddingX(12).Radius(6).Center().
				Cursor(ui.CursorPointer)
			if i == *value {
				item.Background(Alpha(t.Foreground, 0.12))
			}
			item.Children(func() {
				color := t.Muted
				if i == *value {
					color = t.Foreground
				}
				ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(color)
			})
			if item.Clicked() {
				*value = i
				if onChange != nil {
					onChange(i)
				}
				a.Repaint()
			}
		}
	})
}

// themeSource is the mygo appearance an Appearance setting asks for, so
// the window's own chrome follows it too.
func themeSource(appearance int) mygo.ThemeSource {
	switch appearance {
	case AppearanceLight:
		return mygo.ThemeLight
	case AppearanceDark:
		return mygo.ThemeDark
	}
	return mygo.ThemeSystem
}

// StartAppearance gives the window the appearance the settings keep, and
// the theme that matches, before it first shows.
func (a *App) StartAppearance() {
	mygo.Theme.SetSource(themeSource(a.settings.Appearance))
	if a.settings.Appearance == AppearanceSystem {
		// The source is the system's: read it now, as the desktop may
		// be the other way around from when the settings were written.
		a.SetSystemAppearance(mygo.Theme.IsDark())
	}
	a.applyTheme()
}

// setAppearance makes the window follow the desktop, or always light or
// dark, and applies the theme that matches.
func (a *App) setAppearance(n int) {
	a.settings.Appearance = n
	mygo.Theme.SetSource(themeSource(n))
	if n == AppearanceSystem {
		// Back to the system's: read it again, as it may have changed
		// while the window forced light or dark.
		a.systemKnown = false
		a.SetSystemAppearance(mygo.Theme.IsDark())
	}
	a.applyTheme()
}

// stepper draws − 100% +.
func (a *App) stepper(c *ui.Context, label string) {
	t := a.T
	ui.Row(c).Height(30).Padding(2).Radius(8).Background(Alpha(t.Foreground, 0.055)).
		Gap(2).AlignItems(ui.Center).Children(func() {
		minus := ui.Box(c).Width(26).Height(26).Radius(6).Center().Cursor(ui.CursorPointer)
		if minus.Hovered() {
			minus.Background(t.Hover)
		}
		minus.Children(func() { ui.Text(c, "−").FontSize(14).TextColor(t.Muted) })
		if minus.Clicked() {
			z := a.Zoom / 1.1
			if z < 0.5 {
				z = 0.5
			}
			a.Zoom = z
			a.Repaint()
		}
		ui.Text(c, label).FontSize(12.5).TextColor(t.Foreground).Width(52).
			TextAlign(ui.Center)
		plus := ui.Box(c).Width(26).Height(26).Radius(6).Center().Cursor(ui.CursorPointer)
		if plus.Hovered() {
			plus.Background(t.Hover)
		}
		plus.Children(func() { ui.Text(c, "+").FontSize(14).TextColor(t.Muted) })
		if plus.Clicked() {
			z := a.Zoom * 1.1
			if z > 3 {
				z = 3
			}
			a.Zoom = z
			a.Repaint()
		}
	})
}

// settingsButton is a primary, secondary or small button.
func (a *App) settingsButton(c *ui.Context, label string, solid, small bool, onClick func()) {
	t := a.T
	h, pad, size := float32(30), float32(12), float32(12)
	if small {
		h, pad, size = 26, 10, 11.5
	}
	b := ui.Row(c).Height(h).PaddingX(pad).Radius(8).Gap(6).AlignItems(ui.Center)
	if solid {
		b.Background(t.Solid)
	} else {
		b.Background(Alpha(t.Foreground, 0.05)).Border(1, t.Input)
	}
	if onClick != nil {
		b.Cursor(ui.CursorPointer)
		if b.Hovered() && !solid {
			b.Background(t.Hover)
		}
	}
	b.Children(func() {
		color := t.Foreground
		if solid {
			color = t.SolidForeground
		}
		if small {
			color = t.Muted
		}
		ui.Text(c, label).FontSize(size).FontWeight(500).TextColor(color)
	})
	if b.Clicked() && onClick != nil {
		onClick()
	}
}

// settingsLink is a quiet, accent-colored action.
func (a *App) settingsLink(c *ui.Context, label string, onClick func()) {
	t := a.T
	text := ui.Text(c, label).FontSize(12).TextColor(t.Accent)
	if onClick != nil {
		text.Cursor(ui.CursorPointer)
		if text.Clicked() {
			onClick()
		}
	}
}

// settingsDots is the accent color chooser.
func (a *App) settingsDots(c *ui.Context) {
	t := a.T
	colors := []string{"#96b2d7", "#e5c07b", "#f0705f", "#a6bd91", "#b5a2ce"}
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		for _, hex := range colors {
			hex := hex
			wrap := ui.Box(c).Size(24, 24).Radius(12).Center().Cursor(ui.CursorPointer)
			if a.settings.Accent == hex {
				wrap.Border(2, t.Accent)
			}
			wrap.Children(func() {
				ui.Box(c).Size(18, 18).Radius(9).Background(ui.Hex(hex))
			})
			if wrap.Clicked() {
				a.settings.Accent = hex
				a.applyTheme()
			}
		}
		more := ui.Box(c).Size(24, 24).Radius(12).Center().Border(1, t.Input).
			Cursor(ui.CursorPointer)
		more.Children(func() { a.icon(c, "plus", 10, t.Dim) })
	})
}

// settingsLook is the Look section.
func (a *App) settingsLook(c *ui.Context) {
	t := a.T
	a.settingsHead(c, "Look", "Theme, accent and type. Everything applies as you change it.")

	a.settingsCard(c, func() {
		a.settingsRow(c, "Appearance", "Follow the system, or always use light or dark.", func() {
			a.settingsSegmentedOn(c, appearanceNames, &a.settings.Appearance, a.setAppearance)
		})
		a.settingsDivider(c)
		a.settingsRow(c, "Accent", "Focus rings, selections and key hints.", func() {
			a.settingsDots(c)
		})
	})

	a.settingsThemeCard(c)

	a.settingsCard(c, func() {
		a.settingsRow(c, "Interface font", "The panels’ typeface and size.", func() {
			a.settingsFont(c, a.settings.InterfaceFont, a.settings.InterfaceSize, false)
		})
		a.settingsDivider(c)
		a.settingsRow(c, "Terminal font", "Used by every pane, on every machine.", func() {
			a.settingsFont(c, a.settings.TerminalFont, a.settings.TerminalSize, true)
		})
		a.settingsDivider(c)
		a.settingsRow(c, "Zoom", "Scales the whole window, terminals included.", func() {
			a.stepper(c, fmt.Sprintf("%.0f%%", a.Zoom*100))
			a.settingsLink(c, "Reset", func() { a.Zoom = 1; a.Repaint() })
		})
	})

	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		a.icon(c, "file", 12, t.Dim)
		ui.Text(c, a.SettingsPath()).Font(TerminalFont).FontSize(10.5).
			TextColor(t.Muted)
		ui.Text(c, "· saved as you go").FontSize(11).TextColor(t.Dim)
	})
}

// settingsFont is the read-only font name and size the design shows.
func (a *App) settingsFont(c *ui.Context, name string, size int, mono bool) {
	t := a.T
	sel := ui.Row(c).Height(30).MinWidth(150).PaddingX(9).Gap(8).Radius(8).
		Border(1, t.Border).Background(Alpha(t.Foreground, 0.045)).
		AlignItems(ui.Center)
	sel.Children(func() {
		label := ui.Text(c, name).FontSize(12.5).TextColor(t.Foreground)
		if mono {
			label.Font(TerminalFont).FontSize(11.5)
		}
		ui.Spacer(c)
		a.icon(c, "chevron-down", 12, t.Dim)
	})
	ui.Box(c).Size(46, 30).Radius(8).Border(1, t.Border).
		Background(Alpha(t.Foreground, 0.045)).Center().Children(func() {
		ui.Text(c, fmt.Sprintf("%d", size)).Font(TerminalFont).FontSize(11.5).
			TextColor(t.Muted)
	})
}

// settingsThemeCard is the theme picker and its previews.
func (a *App) settingsThemeCard(c *ui.Context) {
	t := a.T
	a.settingsCard(c, func() {
		ui.Column(c).Padding(14, 16, 16, 16).Gap(12).Children(func() {
			ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Start).Children(func() {
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					ui.Text(c, "Theme").FontSize(13.5).FontWeight(500).TextColor(t.Foreground)
					ui.Text(c, "Colors for the chrome and every terminal. The theme picked here wins over a layout’s own.").
						FontSize(11.5).TextColor(t.Dim).Wrap()
				})
				a.settingsLink(c, "Edit colors…", nil)
			})
			ui.Row(c).FillWidth().Gap(12).Children(func() {
				for _, name := range themes {
					a.settingsThemeTile(c, name)
				}
			})
		})
	})
}

// settingsThemeTile is one theme, with a miniature terminal.
func (a *App) settingsThemeTile(c *ui.Context, name string) {
	t := a.T
	th := ThemeByName(name)
	on := a.currentThemeName() == name
	col := ui.Column(c).Grow(1).Gap(6).Cursor(ui.CursorPointer)
	col.Children(func() {
		border := t.Border
		if on {
			border = t.Accent
		} else if !th.Dark {
			border = ui.RGBA(0, 0, 0, 0.14)
		}
		prev := ui.Column(c).Height(82).FillWidth().Radius(8).
			Background(th.Background).Border(1, border).Padding(9, 10).Gap(3)
		prev.Children(func() {
			ui.Row(c).Gap(3).Children(func() {
				for i := 0; i < 3; i++ {
					ui.Box(c).Size(4, 4).Radius(2).Background(th.Dim)
				}
			})
			line := func(s string, color ui.Color) {
				ui.Text(c, s).Font(TerminalFont).FontSize(8).TextColor(color).SingleLine()
			}
			line("$ go test ./...", th.Foreground)
			line("ok  internal/ui", th.Working)
			line("$ hc ls", th.Foreground)
			ui.Box(c).Size(4, 8).Background(th.Accent)
			if on {
				badge := ui.Box(c).Absolute().Top(5).Right(5).Size(15, 15).Radius(8).
					Background(t.Accent).Center()
				badge.Children(func() { a.icon(c, "check", 9, t.AccentForeground) })
			}
		})
		nm := ui.Row(c).Gap(6).AlignItems(ui.Center)
		nm.Children(func() {
			color := t.Muted
			if on {
				color = t.Foreground
			}
			ui.Text(c, strings.Title(name)).FontSize(11.5).TextColor(color)
			if name == "graphite" {
				ui.Text(c, "Default").FontSize(10.5).TextColor(t.Dim)
			}
		})
	})
	if col.Clicked() {
		a.settings.Theme = name
		a.applyTheme()
	}
}

// settingsTerminal is the Terminal section.
func (a *App) settingsTerminal(c *ui.Context) {
	t := a.T
	a.settingsHead(c, "Terminal", "How panes behave, here and on every machine.")

	a.settingsCard(c, func() {
		a.settingsRow(c, "Option as Alt", "Send Option as Alt, or keep it for macOS accents.", func() {
			a.settingsSegmented(c, optionAltNames, &a.settings.OptionAsAlt)
		})
		a.settingsDivider(c)
		a.settingsRow(c, "Clipboard", "Let terminal programs write the clipboard.", func() {
			a.settingsToggle(c, &a.settings.Clipboard)
		})
	})

	a.settingsCard(c, func() {
		a.settingsRow(c, "Minimum contrast", "1.0 leaves program colors alone. Raise it to steady light and dark text.", func() {
			a.settingsSlider(c, &a.settings.MinContrast, 1, 21)
		})
	})

	a.settingsCard(c, func() {
		a.settingsRow(c, "herdr-core", "The background program that runs your terminals and agents.", func() {
			pill := ui.Box(c).Height(28).PaddingX(8).Radius(7).Border(1, t.Border).
				Background(Alpha(t.Foreground, 0.045)).Center()
			pill.Children(func() {
				ui.Text(c, corePath()).Font(TerminalFont).FontSize(11).TextColor(t.Muted).
					Width(230).Ellipsis("…")
			})
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				color := t.Dim
				text := "Not running"
				if a.S != nil {
					color, text = t.Ansi[2], "Connected"
				}
				ui.Box(c).Size(6, 6).Radius(3).Background(color)
				ui.Text(c, text).FontSize(11.5).TextColor(t.Muted)
			})
		})
	})

	a.settingsTerminalPreview(c)
}

// corePath is the core binary the app would use, as a short path.
func corePath() string {
	path, err := exec.LookPath("hc")
	if err != nil {
		return "hc (not on PATH)"
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

// settingsSlider is a horizontal value control.
func (a *App) settingsSlider(c *ui.Context, value *float64, lo, hi float64) {
	t := a.T
	frac := float32((*value - lo) / (hi - lo))
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	ui.Row(c).Width(224).Gap(12).AlignItems(ui.Center).Children(func() {
		track := ui.Box(c).Width(176).Height(20).Cursor(ui.CursorPointer)
		track.Children(func() {
			ui.Box(c).Absolute().Top(8).Left(0).Right(0).Height(4).Radius(2).
				Background(Alpha(t.Foreground, 0.13))
			if frac > 0 {
				ui.Box(c).Absolute().Top(8).Left(0).WidthPercent(frac * 100).Height(4).
					Radius(2).Background(t.Accent)
			}
			ui.Box(c).Absolute().Top(2).LeftPercent(frac*100).Translate(-8, 0).
				Size(16, 16).Radius(8).Background(t.Foreground).Border(1, t.Border)
		})
		ui.Text(c, fmt.Sprintf("%.1f", *value)).Font(TerminalFont).FontSize(11.5).
			TextColor(t.Muted).Width(36).TextAlign(ui.End)
		if x, _, ok := track.PointerPosition(); ok && track.Clicked() {
			f := float64(x / 176)
			if f < 0 {
				f = 0
			}
			if f > 1 {
				f = 1
			}
			*value = lo + f*(hi-lo)
			a.Repaint()
		}
	})
}

// settingsTerminalPreview is a small terminal in the current theme.
func (a *App) settingsTerminalPreview(c *ui.Context) {
	t := a.T
	box := ui.Column(c).FillWidth().Radius(12).Border(1, t.Border).
		Background(t.Background).Clip()
	box.Children(func() {
		ui.Row(c).Height(28).FillWidth().PaddingX(10).Gap(5).AlignItems(ui.Center).
			Children(func() {
				for i := 0; i < 3; i++ {
					ui.Box(c).Size(6, 6).Radius(3).Background(Alpha(t.Foreground, 0.15))
				}
				ui.Text(c, "Preview").FontSize(10.5).TextColor(t.Dim).Padding(0, 0, 0, 6)
			})
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		ui.Column(c).Padding(10, 14).Gap(1).Children(func() {
			a.termLine(c, "~", ` echo "agents keep running"`, t.Accent, t.TerminalForeground)
			a.termLine(c, "", "agents keep running", t.Dim, t.TerminalForeground)
			a.termLine(c, "~", " hc ls", t.Accent, t.TerminalForeground)
			a.termLine(c, "", "t3  claude    working    herdr-mygo", t.Muted, t.Muted)
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "~").Font(TerminalFont).FontSize(10.5).TextColor(t.Accent)
				ui.Box(c).Size(6, 11).Background(t.Accent)
			})
		})
	})
}

func (a *App) termLine(c *ui.Context, prompt, body string, promptColor, bodyColor ui.Color) {
	ui.Row(c).Gap(4).Children(func() {
		if prompt != "" {
			ui.Text(c, prompt).Font(TerminalFont).FontSize(10.5).TextColor(promptColor)
		}
		ui.Text(c, body).Font(TerminalFont).FontSize(10.5).TextColor(bodyColor).SingleLine()
	})
}

// settingsMachines is the Machines section.
func (a *App) settingsMachines(c *ui.Context) {
	a.settingsHead(c, "Machines", "Herdr reaches other computers over ssh. Their terminals and agents keep running there.")

	var machines []*herdr.Machine
	if a.S != nil {
		machines = a.S.Machines()
	}
	a.settingsCard(c, func() {
		a.settingsMachineRow(c, "monitor", "This computer", "where this window runs", "connected", "", nil)
		if len(machines) > 0 {
			for _, m := range machines {
				m := m
				a.settingsDivider(c)
				why := ""
				if m.Why != nil {
					why = *m.Why
				}
				a.settingsMachineRow(c, "server", m.Name, m.Target, m.State, why, func() {
					a.settingsMachineActions(c, m)
				})
			}
			a.settingsDivider(c)
			a.settingsAddMachineRow(c)
		}
	})

	if len(machines) == 0 {
		a.settingsEmptyMachines(c)
	} else {
		a.settingsCard(c, func() {
			a.settingsRow(c, "Keep SSH connections warm", "Adds keepalives and reuses connections. Off leaves your ssh config untouched.", func() {
				a.settingsToggle(c, &a.settings.KeepSSHWarm)
			})
		})
	}

	a.settingsNote(c, "server", "A machine runs its own herdr-core, so its agents keep working when this Mac sleeps.")
}

// settingsEmptyMachines is the first-run Machines page.
func (a *App) settingsEmptyMachines(c *ui.Context) {
	t := a.T
	a.settingsCard(c, func() {
		ui.Column(c).Padding(30, 20).Gap(9).AlignItems(ui.Center).Children(func() {
			icon := ui.Box(c).Size(46, 46).Radius(12).Background(Alpha(t.Foreground, 0.055)).
				Center()
			icon.Children(func() { a.icon(c, "server", 20, t.Muted) })
			ui.Text(c, "No other machines yet").FontSize(13.5).FontWeight(500).
				TextColor(t.Foreground)
			ui.Text(c, "Add a server or a Mac from your ~/.ssh/config, or type how ssh reaches it. Its terminals and agents run there, and keep working when this Mac sleeps.").
				FontSize(11.5).TextColor(t.Dim).Wrap().TextAlign(ui.Center).MaxWidth(430)
			a.settingsButton(c, "Add machine", true, false, func() {
				a.OpenOverlay("machine-add")
			})
		})
	})
}

// settingsMachineRow draws one machine and its status.
func (a *App) settingsMachineRow(c *ui.Context, icon, name, target, state, why string, actions func()) {
	t := a.T
	row := ui.Row(c).FillWidth().MinHeight(66).Padding(11, 16).Gap(12).
		AlignItems(ui.Center)
	row.Children(func() {
		tile := ui.Box(c).Size(34, 34).Radius(9).Background(Alpha(t.Foreground, 0.055)).
			Center()
		tile.Children(func() { a.icon(c, icon, 16, t.Muted) })
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, name).FontSize(13.5).FontWeight(500).TextColor(t.Foreground)
			ui.Text(c, target).FontSize(11.5).TextColor(t.Dim).Ellipsis("…")
			if why != "" {
				ui.Text(c, why).FontSize(11).TextColor(t.Blocked).Ellipsis("…")
			}
		})
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(6, 6).Radius(3).Background(a.machineStateColor(state))
			ui.Text(c, machineStateText(state)).FontSize(11.5).TextColor(t.Muted)
		})
		if actions != nil {
			actions()
		}
	})
}

func (a *App) machineStateColor(state string) ui.Color {
	switch state {
	case "connected":
		return a.T.Ansi[2]
	case "connecting":
		return a.T.Working
	case "needs_login", "unreachable":
		return a.T.Blocked
	}
	return a.T.Dim
}

func machineStateText(state string) string {
	switch state {
	case "connected":
		return "Connected"
	case "connecting":
		return "Connecting"
	case "needs_login":
		return "Needs login"
	case "unreachable":
		return "Offline"
	}
	return state
}

// settingsMachineActions is Connect or Remove for one machine.
func (a *App) settingsMachineActions(c *ui.Context, m *herdr.Machine) {
	if m.Connected() {
		a.settingsButton(c, "Remove", false, true, func() {
			go func() {
				if err := a.S.RemoveMachine(m.Name); err != nil {
					a.toast("Cannot remove the machine", err.Error(), "error")
				}
				a.S.RefreshMachines()
				a.Repaint()
			}()
		})
		return
	}
	a.settingsButton(c, "Connect", false, true, func() {
		go func() {
			if err := a.S.ConnectMachine(m.Name); err != nil {
				a.toast("Cannot connect", err.Error(), "error")
			}
			a.Repaint()
		}()
	})
}

// settingsAddMachineRow opens the add-machine sheet.
func (a *App) settingsAddMachineRow(c *ui.Context) {
	t := a.T
	row := ui.Row(c).FillWidth().Height(48).PaddingX(16).Gap(10).
		AlignItems(ui.Center).Cursor(ui.CursorPointer)
	if row.Hovered() {
		row.Background(t.Hover)
	}
	row.Children(func() {
		plus := ui.Box(c).Size(24, 24).Radius(7).Border(1, t.Input).Center()
		plus.Children(func() { a.icon(c, "plus", 12, t.Muted) })
		ui.Text(c, "Add machine").FontSize(12.5).TextColor(t.Foreground)
		ui.Spacer(c)
		ui.Text(c, "From ~/.ssh/config or an address").FontSize(11).TextColor(t.Dim).
			Ellipsis("…")
	})
	if row.Clicked() {
		a.OpenOverlay("machine-add")
	}
}

// settingsAlerts is the Alerts section.
func (a *App) settingsAlerts(c *ui.Context) {
	t := a.T
	a.settingsHead(c, "Alerts", "When an agent finishes or needs you, Herdr can play a sound and post a toast.")

	a.settingsCard(c, func() {
		a.settingsRow(c, "Done", "A sound and a toast when an agent finishes while you are elsewhere.", func() {
			a.settingsToggle(c, &a.settings.AlertDone)
		})
		a.settingsDivider(c)
		a.settingsRow(c, "Needs you", "When an agent asks a question or waits for approval.", func() {
			a.settingsToggle(c, &a.settings.AlertNeeds)
		})
	})

	soundOn := a.settings.AlertSound
	a.settingsCard(c, func() {
		a.settingsRow(c, "Play a sound", "Use the built-in sounds, or choose your own files.", func() {
			a.settingsToggle(c, &a.settings.AlertSound)
		})
		a.settingsDivider(c)
		a.settingsRowWhen(c, soundOn, "Done sound", "Plays when an agent finishes.", func() {
			a.settingsSound(c, "Glass")
		})
		a.settingsDivider(c)
		a.settingsRowWhen(c, soundOn, "Needs you sound", "Plays when an agent asks something.", func() {
			a.settingsSound(c, "Ping")
		})
	})

	a.settingsCard(c, func() {
		a.settingsRowWhen(c, soundOn, "By agent", "Turn a sound off for one agent without changing the rest.", func() {
			ui.Text(c, "All agents on").FontSize(11.5).TextColor(t.Muted)
			a.settingsLink(c, "Customize  ›", nil)
		})
	})

	note := "Sounds play only when the agent is in a space you are not looking at."
	if !soundOn {
		note = "Sound is off. Toasts still appear when an agent finishes or needs you."
	}
	a.settingsNote(c, "bell", note)
}

// settingsSound is the preview and file for one alert sound.
func (a *App) settingsSound(c *ui.Context, name string) {
	t := a.T
	play := ui.Box(c).Size(26, 26).Radius(13).Border(1, t.Input).Center()
	play.Children(func() { a.icon(c, "play", 10, t.Muted) })
	ui.Text(c, name).FontSize(12).TextColor(t.Muted).Width(54)
	a.settingsButton(c, "Choose…", false, true, nil)
}

// settingsUpdates is the Updates section.
func (a *App) settingsUpdates(c *ui.Context) {
	t := a.T
	a.settingsHead(c, "Updates", "Herdr updates itself. herdr-core updates alongside, without ending a terminal.")

	a.settingsCard(c, func() {
		ui.Row(c).FillWidth().Padding(16).Gap(14).AlignItems(ui.Center).Children(func() {
			a.logo(c, 38)
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Text(c, "Herdr "+settingsVersion).FontSize(15).FontWeight(600).
					TextColor(t.Foreground)
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(6, 6).Radius(3).Background(t.Ansi[2])
					ui.Text(c, "Up to date").FontSize(11.5).TextColor(t.Muted)
				})
			})
			a.settingsButton(c, "Check now", false, false, nil)
		})
		a.settingsDivider(c)
		a.settingsRow(c, "What’s new", "Release notes for the version you are running.", func() {
			a.settingsLink(c, "View notes  ›", nil)
		})
	})

	a.settingsCard(c, func() {
		a.settingsRow(c, "Channel", "Preview gets releases early. Homebrew installs stay on stable.", func() {
			a.settingsSegmented(c, updateChannelNames, &a.settings.UpdateChannel)
		})
		a.settingsDivider(c)
		a.settingsRow(c, "Install automatically", "Download and install when Herdr is not running.", func() {
			a.settingsToggle(c, &a.settings.AutoInstall)
		})
		a.settingsDivider(c)
		a.settingsRow(c, "Check in the background", "Once a day, without interrupting a terminal.", func() {
			a.settingsToggle(c, &a.settings.BackgroundCheck)
		})
	})

	a.settingsCard(c, func() {
		a.settingsRow(c, "herdr-core", "Updates with Herdr. Terminals keep running through it.", func() {
			color, text := t.Dim, "Not running"
			if a.S != nil {
				color, text = t.Ansi[2], "Connected"
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(6, 6).Radius(3).Background(color)
				ui.Text(c, text).FontSize(11.5).TextColor(t.Muted)
			})
		})
	})

	a.settingsNote(c, "refresh-cw", "Source builds never check for updates.")
}

// settingsPrivacy is the Privacy section.
func (a *App) settingsPrivacy(c *ui.Context) {
	t := a.T
	a.settingsHead(c, "Privacy", "What Herdr sends, and what it never touches.")

	a.settingsCard(c, func() {
		a.settingsRow(c, "Anonymous usage counts", "A daily count of versions, machines, agents, spaces, tabs and commands. No terminal content, file names or folder names.", func() {
			a.settingsToggle(c, &a.settings.Telemetry)
		})
		a.settingsDivider(c)
		a.settingsRow(c, "Reports", "At most one a day. Nothing comes from herdr-core, and nothing is sent from a source build.", func() {
			a.settingsButton(c, "View the report", false, true, nil)
		})
	})

	a.settingsCard(c, func() {
		ui.Column(c).Padding(14, 16, 14, 16).Gap(10).Children(func() {
			ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Start).Children(func() {
				ui.Column(c).Grow(1).Gap(2).Children(func() {
					ui.Text(c, "What would be sent").FontSize(13.5).FontWeight(500).TextColor(t.Foreground)
					ui.Text(c, "A report like this goes out at most once a day. This is the whole file.").
						FontSize(11.5).TextColor(t.Dim).Wrap()
				})
				a.settingsLink(c, "Copy", nil)
			})
			block := ui.Box(c).FillWidth().MinHeight(150).Radius(9).Border(1, t.Border).
				Background(Alpha(t.Foreground, 0.045)).Padding(12, 14)
			block.Children(func() {
				ui.Text(c, a.settingsReport()).Font(TerminalFont).FontSize(10.5).
					TextColor(t.Muted).Wrap()
			})
			a.settingsNote(c, "shield", "Terminal text, names and herdr-core data are never part of it.")
		})
	})

	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		a.icon(c, "file", 12, t.Dim)
		ui.Text(c, "DO_NOT_TRACK=1").Font(TerminalFont).FontSize(10.5).TextColor(t.Muted)
		ui.Text(c, "or").FontSize(11).TextColor(t.Dim)
		ui.Text(c, "HERDR_TELEMETRY=0").Font(TerminalFont).FontSize(10.5).TextColor(t.Muted)
		ui.Text(c, "turns it off for one run.").FontSize(11).TextColor(t.Dim)
	})
}

// settingsReport is the report the Privacy section would send, from this
// session's own counts.
func (a *App) settingsReport() string {
	agents, spaces, tabs, machines := 0, 0, 0, 0
	if a.S != nil {
		agents = len(a.S.Agents())
		spaces = len(a.S.Places())
		tabs = len(a.S.Tabs())
		machines = len(a.S.Machines())
	}
	return fmt.Sprintf(`{
  "version": "%s",
  "os": "%s %s",
  "agents": %d,
  "spaces": %d,
  "tabs": %d,
  "machines": %d,
  "layout": "main"
}`, settingsVersion, runtime.GOOS, runtime.GOARCH, agents, spaces, tabs, machines)
}
