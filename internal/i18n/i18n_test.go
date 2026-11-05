package i18n

import (
	"encoding/json"
	native "github.com/egoist/mygo/i18n"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLanguageSelection(t *testing.T) {
	for _, tc := range []struct {
		preferences []string
		want        string
	}{
		{nil, "en"}, {[]string{"en-GB"}, "en"}, {[]string{"zh_CN.UTF-8"}, "zh"}, {[]string{"zh-Hant-TW"}, "zh-Hant"}, {[]string{"ja_JP.UTF-8"}, "ja"}, {[]string{"fr-FR"}, "en"}, {[]string{"de-DE", "ja-JP", "en-US"}, "ja"}, {[]string{"C", "POSIX"}, "en"}, {[]string{"invalid_locale", "zh-CN"}, "zh"},
	} {
		t.Run(strings.Join(tc.preferences, "/"), func(t *testing.T) {
			if got := Resolve(tc.preferences); got != tc.want {
				t.Fatalf("Resolve(%q)=%q, want %q", tc.preferences, got, tc.want)
			}
		})
	}
}

func TestTranslationsAndFallback(t *testing.T) {
	for _, tc := range []struct{ locale, want string }{{"en", "Settings"}, {"zh-CN", "设置"}, {"ja-JP", "設定"}, {"fr-FR", "Settings"}} {
		l := New(tc.locale)
		if got := l.Text("Settings"); got != tc.want {
			t.Errorf("%s: %q", tc.locale, got)
		}
		if got := l.Text("An unknown message"); got != "An unknown message" {
			t.Errorf("missing message: %q", got)
		}
	}
	if got := New("ja").Text("%d of %d", 2, 7); got != "7 件中 2 件目" {
		t.Errorf("format arguments: %q", got)
	}
	if got := New("zh").Text("Nothing matches %q.", "测试"); !strings.Contains(got, "测试") || strings.Contains(got, "%!") {
		t.Errorf("format: %q", got)
	}
}

func TestCatalogsCompleteAndFormatsValid(t *testing.T) {
	var source map[string]json.RawMessage
	data, err := os.ReadFile(filepath.Join("locales", "en.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	for _, locale := range []string{"zh", "zh-Hant", "ja"} {
		var translated map[string]json.RawMessage
		data, err := os.ReadFile(filepath.Join("locales", locale+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &translated); err != nil {
			t.Fatal(err)
		}
		for key := range source {
			if strings.HasPrefix(key, "@") {
				continue
			}
			var text string
			if err := json.Unmarshal(translated[key], &text); err != nil || strings.TrimSpace(text) == "" {
				t.Errorf("%s missing %s", locale, key)
				continue
			}
			if _, err := native.ParseMessage(text); err != nil {
				t.Errorf("%s invalid %s: %v", locale, key, err)
			}
		}
	}
}

func TestSourceMessagesResolveToCatalogKeys(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("locales", "en.json"))
	if err != nil {
		t.Fatal(err)
	}
	var source map[string]json.RawMessage
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	for text, key := range messageIDs {
		if _, ok := source[key]; !ok {
			t.Errorf("%q maps to missing key %q", text, key)
		}
		if !strings.Contains(text, "%") {
			if got := New("en").Text(text); got != text {
				t.Errorf("%q resolved to %q", text, got)
			}
		}
	}
}

func TestEnglishPluralForms(t *testing.T) {
	l := New("en")
	if got := l.Text("%d agents", 1); got != "1 agent" {
		t.Fatal(got)
	}
	if got := l.Text("%d agents", 2); got != "2 agents" {
		t.Fatal(got)
	}
}
