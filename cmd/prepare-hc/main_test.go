package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadVerifiesChecksum(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("release"))
	}))
	defer server.Close()
	sum := sha256.Sum256([]byte("release"))
	a := artifact{URL: server.URL, SHA256: fmt.Sprintf("%x", sum)}
	if _, err := download(server.Client(), a); err != nil {
		t.Fatal(err)
	}
	a.SHA256 = strings.Repeat("0", 64)
	if _, err := download(server.Client(), a); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("corrupt download accepted: %v", err)
	}
}

func TestUnpackRejectsTraversal(t *testing.T) {
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	file, err := writer.Create("../outside")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write([]byte("escape"))
	_ = writer.Close()
	if err := unpack(t.TempDir(), data.Bytes(), true); err == nil {
		t.Fatal("archive path traversal accepted")
	}
}

func TestStageWindowsIncludesConPTYAndNotices(t *testing.T) {
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	files := map[string]string{
		"herdr-core.exe": "executable", "conpty/conpty.dll": "dll",
		"conpty/x64/OpenConsole.exe": "console", "LICENSE": "license",
		"THIRD-PARTY-NOTICES/core.txt": "notices",
	}
	for name, contents := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = file.Write([]byte(contents))
	}
	_ = writer.Close()
	dir := t.TempDir()
	if err := stageCore(dir, "windows/amd64", data.Bytes()); err != nil {
		t.Fatal(err)
	}
	files["hc.exe"] = "executable"
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(got) != want {
			t.Fatalf("%s = %q, %v; want %q", name, got, err, want)
		}
	}
}

func TestStageRejectsIncompleteWindowsArchive(t *testing.T) {
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	file, _ := writer.Create("herdr-core.exe")
	_, _ = file.Write([]byte("executable"))
	_ = writer.Close()
	if err := stageCore(t.TempDir(), "windows/amd64", data.Bytes()); err == nil {
		t.Fatal("Windows core without ConPTY accepted")
	}
}

func TestStageMacOSPreservesCoreBundle(t *testing.T) {
	var data bytes.Buffer
	gz := gzip.NewWriter(&data)
	writer := tar.NewWriter(gz)
	for name, contents := range map[string]string{
		"herdr-core": "standalone", "Herdr Core.app/Contents/MacOS/herdr-core": "bundled",
		"Herdr Core.app/Contents/Info.plist": "plist", "LICENSE": "license",
	} {
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(contents))}); err != nil {
			t.Fatal(err)
		}
		_, _ = writer.Write([]byte(contents))
	}
	_ = writer.Close()
	_ = gz.Close()
	dir := t.TempDir()
	if err := stageCore(dir, "darwin/arm64", data.Bytes()); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "hc"))
	if err != nil || string(got) != "bundled" {
		t.Fatalf("hc = %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Herdr Core.app/Contents/Info.plist")); err != nil {
		t.Fatal("core bundle lost:", err)
	}
}

func TestUnsupportedPlatformFailsBeforeStaging(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "resources")
	if err := prepare(dir, []string{"darwin/arm64", "windows/arm64"}); err == nil {
		t.Fatal("unsupported native Windows ARM64 accepted")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("unsupported target changed resources: %v", err)
	}
}
