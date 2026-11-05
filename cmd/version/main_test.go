package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func versionFixture(t *testing.T, version string) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.WriteFile("mygo.json", []byte(`{"name":"Herdr","version":"`+version+`","macos":{"minimumSystemVersion":"13.0"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"GITHUB_REF_TYPE", "GITHUB_REF_NAME", "GITHUB_RUN_NUMBER", "GITHUB_SHA", "GITHUB_OUTPUT"} {
		t.Setenv(key, "")
	}
}

func TestBumpVersion(t *testing.T) {
	for _, tc := range []struct{ part, want string }{
		{"patch", "0.2.4"}, {"minor", "0.3.0"}, {"major", "1.0.0"},
	} {
		t.Run(tc.part, func(t *testing.T) {
			versionFixture(t, "0.2.3-rc.1")
			var out bytes.Buffer
			if err := run([]string{"bump", tc.part}, &out); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile("mygo.json")
			if err != nil {
				t.Fatal(err)
			}
			var config struct {
				Name, Version string
				MacOS         map[string]string
			}
			if err := json.Unmarshal(data, &config); err != nil {
				t.Fatal(err)
			}
			if config.Version != tc.want || config.Name != "Herdr" || config.MacOS["minimumSystemVersion"] != "13.0" {
				t.Fatalf("bump changed wrong fields: %s", data)
			}
			if out.String() != tc.want+"\n" {
				t.Fatalf("output = %q", out.String())
			}
		})
	}
}

func TestSetVersionRejectsInvalidInputWithoutWriting(t *testing.T) {
	for _, version := range []string{"v1.2.3", "01.2.3", "1.2", "1.2.3-01", "1.2.3-rc..1", "1.2.3+metadata", "1.2.3\ninjected=true"} {
		t.Run(version, func(t *testing.T) {
			versionFixture(t, "0.1.0")
			before, _ := os.ReadFile("mygo.json")
			if err := run([]string{"set", version}, &bytes.Buffer{}); err == nil {
				t.Fatal("invalid version accepted")
			}
			after, _ := os.ReadFile("mygo.json")
			if !bytes.Equal(before, after) {
				t.Fatal("invalid input modified mygo.json")
			}
		})
	}
}

func TestSetPrereleaseVersion(t *testing.T) {
	versionFixture(t, "0.1.0")
	var out bytes.Buffer
	if err := run([]string{"set", "0.2.0-rc.1"}, &out); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := run(nil, &out); err != nil || out.String() != "0.2.0-rc.1\n" {
		t.Fatalf("read after set = %q, %v", out.String(), err)
	}
}

func TestCIVersion(t *testing.T) {
	for _, tc := range []struct {
		name, base, refType, ref, want string
		wantError                      bool
	}{
		{"push", "0.1.0", "branch", "master", "version=0.1.0-dev.42.gabcdef1\ntag=v0.1.0-dev.42.gabcdef1\nprerelease=true\n", false},
		{"preview base", "0.2.0-rc.1", "branch", "master", "version=0.2.0-rc.1.dev.42.gabcdef1\ntag=v0.2.0-rc.1.dev.42.gabcdef1\nprerelease=true\n", false},
		{"stable tag", "0.1.0", "tag", "v0.1.0", "version=0.1.0\ntag=v0.1.0\nprerelease=false\n", false},
		{"preview tag", "0.2.0-rc.1", "tag", "v0.2.0-rc.1", "version=0.2.0-rc.1\ntag=v0.2.0-rc.1\nprerelease=true\n", false},
		{"mismatched tag", "0.1.0", "tag", "v0.2.0", "", true},
		{"invalid base", "0.1", "branch", "master", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			versionFixture(t, tc.base)
			t.Setenv("GITHUB_REF_TYPE", tc.refType)
			t.Setenv("GITHUB_REF_NAME", tc.ref)
			t.Setenv("GITHUB_RUN_NUMBER", "42")
			t.Setenv("GITHUB_SHA", "abcdef1234567890abcdef1234567890abcdef12")
			t.Setenv("GITHUB_OUTPUT", "outputs")
			before, _ := os.ReadFile("mygo.json")
			var out bytes.Buffer
			err := run([]string{"ci"}, &out)
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, wantError = %v", err, tc.wantError)
			}
			if out.String() != tc.want {
				t.Fatalf("metadata = %q, want %q", out.String(), tc.want)
			}
			output, _ := os.ReadFile("outputs")
			if string(output) != tc.want {
				t.Fatalf("GitHub outputs = %q, want %q", output, tc.want)
			}
			after, _ := os.ReadFile("mygo.json")
			if !bytes.Equal(before, after) {
				t.Fatal("ci changed the checked-in base version")
			}
		})
	}
}

func TestCIRequiresBuildIdentity(t *testing.T) {
	for _, tc := range []struct{ run, sha string }{
		{"", "abcdef1234567890abcdef1234567890abcdef12"},
		{"42", ""}, {"0", "abcdef1234567890abcdef1234567890abcdef12"},
		{"42\ninjected=true", "abcdef1234567890abcdef1234567890abcdef12"},
	} {
		t.Run(tc.run+tc.sha, func(t *testing.T) {
			versionFixture(t, "0.1.0")
			t.Setenv("GITHUB_REF_TYPE", "branch")
			t.Setenv("GITHUB_RUN_NUMBER", tc.run)
			t.Setenv("GITHUB_SHA", tc.sha)
			var out bytes.Buffer
			if err := run([]string{"ci"}, &out); err == nil || strings.Contains(out.String(), "tag=") {
				t.Fatalf("missing build identity produced release metadata: %q, %v", out.String(), err)
			}
		})
	}
}
