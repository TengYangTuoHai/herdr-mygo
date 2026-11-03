package ui

import "testing"

func TestDefaultSettings(t *testing.T) {
	s := defaultSettings()
	if s.Section != "look" {
		t.Fatalf("section = %q, want look", s.Section)
	}
	if s.Theme != "graphite" {
		t.Fatalf("theme = %q, want graphite", s.Theme)
	}
	if s.InterfaceSize != 13 || s.TerminalSize != 14 {
		t.Fatalf("font sizes = %d/%d, want 13/14", s.InterfaceSize, s.TerminalSize)
	}
	if !s.Clipboard || !s.AlertDone || !s.AlertNeeds || !s.AlertSound || !s.Telemetry {
		t.Fatalf("defaults should turn clipboard, alerts and telemetry on: %+v", s)
	}
}

func TestSettingsResultsMatchThemeFirst(t *testing.T) {
	a := &App{T: ThemeByName("graphite")}
	a.settings = defaultSettings()
	a.Zoom = 1
	a.settings.Search = "theme"

	results := a.settingsResults()
	if len(results) == 0 {
		t.Fatal("no results for theme")
	}
	if results[0].Title != "Theme" {
		t.Fatalf("first result = %q, want Theme", results[0].Title)
	}
}

func TestSettingsResultsEmptyQueryListsEveryEntry(t *testing.T) {
	a := &App{T: ThemeByName("graphite")}
	a.settings = defaultSettings()
	a.Zoom = 1

	if got, want := len(a.settingsResults()), len(a.settingsEntries()); got != want {
		t.Fatalf("results = %d, want %d", got, want)
	}
}

func TestChooseSettingsResultOpensItsSection(t *testing.T) {
	a := &App{T: ThemeByName("graphite")}
	a.settings = defaultSettings()
	a.Zoom = 1
	a.settings.Search = "sound"
	a.settings.SearchOpen = true

	at := -1
	for i, r := range a.settingsResults() {
		if r.Section == "alerts" {
			at = i
			break
		}
	}
	if at < 0 {
		t.Fatal("no alerts result for sound")
	}
	a.chooseSettingsResult(at)
	if a.settings.Section != "alerts" {
		t.Fatalf("section = %q, want alerts", a.settings.Section)
	}
	if a.settings.SearchOpen {
		t.Fatal("search should close after choosing")
	}
}

func TestSettingsMoveStopsAtEnds(t *testing.T) {
	a := &App{T: ThemeByName("graphite")}
	a.settings = defaultSettings()
	a.Zoom = 1
	a.settings.SearchAt = 0

	a.settingsMove(-1)
	if a.settings.SearchAt != 0 {
		t.Fatalf("moved above the first result to %d", a.settings.SearchAt)
	}
	a.settingsMove(1)
	if a.settings.SearchAt != 1 {
		t.Fatalf("moved to %d, want 1", a.settings.SearchAt)
	}
	results := len(a.settingsResults())
	a.settings.SearchAt = results - 1
	a.settingsMove(1)
	if a.settings.SearchAt != results-1 {
		t.Fatalf("moved past the last result to %d", a.settings.SearchAt)
	}
}
