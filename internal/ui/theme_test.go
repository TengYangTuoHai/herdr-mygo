package ui

import "testing"

// TestThemeForFollowsTheAppearance picks paper in the light and graphite
// in the dark when the theme itself is the other way around.
func TestThemeForFollowsTheAppearance(t *testing.T) {
	cases := []struct {
		name       string
		appearance int
		systemDark bool
		want       string
	}{
		{"graphite", AppearanceSystem, true, "graphite"},
		{"graphite", AppearanceSystem, false, "paper"},
		{"graphite", AppearanceLight, true, "paper"},
		{"graphite", AppearanceDark, false, "graphite"},
		{"ink", AppearanceLight, true, "paper"},
		{"moss", AppearanceSystem, true, "moss"},
		{"paper", AppearanceSystem, true, "graphite"},
		{"paper", AppearanceSystem, false, "paper"},
	}
	for _, c := range cases {
		name, th := ThemeFor(c.name, c.appearance, c.systemDark)
		if name != c.want {
			t.Errorf("ThemeFor(%q, %d, %v) = %q, want %q", c.name, c.appearance, c.systemDark, name, c.want)
		}
		wantDark := c.systemDark
		switch c.appearance {
		case AppearanceLight:
			wantDark = false
		case AppearanceDark:
			wantDark = true
		}
		if th.Dark != wantDark {
			t.Errorf("ThemeFor(%q, %d, %v).Dark = %v, want %v", c.name, c.appearance, c.systemDark, th.Dark, wantDark)
		}
	}
}

// TestApplyThemeKeepsTheAccentPicked draws the accent the settings keep,
// whichever theme the appearance then picks.
func TestApplyThemeKeepsTheAccentPicked(t *testing.T) {
	app := NewApp(nil, ThemeByName("graphite"))
	app.settings = defaultSettings()
	app.settings.Accent = "#ff0000"
	app.applyTheme()
	for _, name := range []string{"graphite", "paper"} {
		app.settings.Theme = name
		app.applyTheme()
		if c := app.T.Accent; c.R != 255 || c.G != 0 || c.B != 0 {
			t.Fatalf("%s accent = %+v, want red", name, c)
		}
	}
}

// TestApplyThemeFollowsTheSystem re-themes a window set to follow the
// desktop when the desktop goes light or dark.
func TestApplyThemeFollowsTheSystem(t *testing.T) {
	app := NewApp(nil, ThemeByName("graphite"))
	app.settings = defaultSettings()

	app.SetSystemAppearance(true)
	if !app.T.Dark || app.currentThemeName() != "graphite" {
		t.Fatalf("dark system drew %q (dark %v), want graphite", app.currentThemeName(), app.T.Dark)
	}
	app.SetSystemAppearance(false)
	if app.T.Dark || app.currentThemeName() != "paper" {
		t.Fatalf("light system drew %q (dark %v), want paper", app.currentThemeName(), app.T.Dark)
	}

	// A theme picked stays, once the appearance asks for the system's:
	// ink is dark, so a light desktop still draws paper.
	app.settings.Theme = "ink"
	app.applyTheme()
	if app.T.Dark {
		t.Fatalf("ink under a light desktop drew a dark theme")
	}
	app.settings.Appearance = AppearanceDark
	app.applyTheme()
	if !app.T.Dark || app.currentThemeName() != "ink" {
		t.Fatalf("ink forced dark drew %q (dark %v), want ink", app.currentThemeName(), app.T.Dark)
	}
}
