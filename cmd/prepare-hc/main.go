// prepare-hc stages verified upstream releases in MyGo's platform resources.
package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed release.json
var releaseJSON []byte

var targets = map[string]string{
	"darwin/arm64": "macos-aarch64", "darwin/amd64": "macos-x86_64",
	"linux/arm64": "linux-aarch64", "linux/amd64": "linux-x86_64",
	"windows/amd64": "windows-x86_64",
}

type artifact struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type release struct {
	Version string              `json:"version"`
	Targets map[string]artifact `json:"targets"`
}

func main() {
	platforms := flag.String("platform", "darwin/arm64,darwin/amd64,windows/amd64,linux/arm64,linux/amd64", "comma-separated OS/architecture targets (upstream has no Windows ARM64 build)")
	resources := flag.String("resources", "resources", "MyGo resources directory, relative to the working directory")
	flag.Parse()
	if err := prepare(*resources, strings.Split(*platforms, ",")); err != nil {
		fmt.Fprintln(os.Stderr, "prepare-hc:", err)
		os.Exit(1)
	}
}

func prepare(resources string, platforms []string) error {
	var manifest release
	if err := json.Unmarshal(releaseJSON, &manifest); err != nil {
		return err
	}
	// Validate the whole request before changing any platform's resources.
	for _, platform := range platforms {
		if targets[platform] == "" {
			return fmt.Errorf("no upstream hc release for %q; supported: darwin/arm64,darwin/amd64,windows/amd64,linux/arm64,linux/amd64", platform)
		}
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	for _, platform := range platforms {
		a := manifest.Targets[targets[platform]]
		data, err := download(client, a)
		if err != nil {
			return fmt.Errorf("%s: %w", platform, err)
		}
		platformDir := filepath.Join(resources, strings.ReplaceAll(platform, "/", "-"))
		if err := os.MkdirAll(platformDir, 0o755); err != nil {
			return err
		}
		stage, err := os.MkdirTemp(platformDir, ".hc-")
		if err != nil {
			return err
		}
		err = stageCore(stage, platform, data)
		if err == nil {
			// bin is owned by this tool, so ConPTY and notices update together.
			err = os.RemoveAll(filepath.Join(platformDir, "bin"))
		}
		if err == nil {
			err = os.Rename(stage, filepath.Join(platformDir, "bin"))
		}
		_ = os.RemoveAll(stage)
		if err != nil {
			return fmt.Errorf("%s: %w", platform, err)
		}
		fmt.Printf("hc %s staged in %s/bin\n", manifest.Version, platformDir)
	}
	return nil
}

func download(client *http.Client, a artifact) ([]byte, error) {
	resp, err := client.Get(a.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", a.URL, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 128<<20))
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != a.SHA256 {
		return nil, fmt.Errorf("checksum mismatch for %s", a.URL)
	}
	return data, nil
}

func stageCore(dir, platform string, data []byte) error {
	windows := strings.HasPrefix(platform, "windows/")
	if err := unpack(dir, data, windows); err != nil {
		return err
	}
	canonical, alias := "herdr-core", "hc"
	if windows {
		canonical += ".exe"
		alias += ".exe"
		for _, name := range []string{"conpty/conpty.dll", "conpty/x64/OpenConsole.exe"} {
			if info, err := os.Stat(filepath.Join(dir, name)); err != nil || !info.Mode().IsRegular() {
				return fmt.Errorf("release is missing %s", name)
			}
		}
	}
	if strings.HasPrefix(platform, "darwin/") {
		canonical = "Herdr Core.app/Contents/MacOS/herdr-core"
	}
	data, err := os.ReadFile(filepath.Join(dir, canonical))
	if err != nil {
		return fmt.Errorf("release is missing %s: %w", canonical, err)
	}
	// Copy rather than link: Windows builders need no symlink privileges.
	return os.WriteFile(filepath.Join(dir, alias), data, 0o755)
}

func unpack(dir string, data []byte, zipped bool) error {
	if zipped {
		reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return err
		}
		for _, file := range reader.File {
			if !file.Mode().IsRegular() && !file.FileInfo().IsDir() {
				return fmt.Errorf("unsupported archive entry %q", file.Name)
			}
			input, err := file.Open()
			if err != nil {
				return err
			}
			err = extract(dir, file.Name, file.Mode(), input)
			_ = input.Close()
			if err != nil {
				return err
			}
		}
		return nil
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
			return fmt.Errorf("unsupported archive entry %q", header.Name)
		}
		if err := extract(dir, header.Name, header.FileInfo().Mode(), reader); err != nil {
			return err
		}
	}
}

func extract(dir, name string, mode os.FileMode, input io.Reader) error {
	if !filepath.IsLocal(name) || strings.Contains(name, "\\") {
		return fmt.Errorf("invalid archive path %q", name)
	}
	path := filepath.Join(dir, filepath.FromSlash(name))
	if mode.IsDir() {
		return os.MkdirAll(path, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	_, err = io.Copy(out, input)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}
