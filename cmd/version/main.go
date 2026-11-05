// version keeps MyGo packages and GitHub releases on the same version.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$`)
	numeric        = regexp.MustCompile(`^[0-9]+$`)
	runPattern     = regexp.MustCompile(`^[1-9][0-9]*$`)
	shaPattern     = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validate(version string) error {
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("invalid version %q: use MAJOR.MINOR.PATCH with an optional prerelease (no v prefix or build metadata)", version)
	}
	if _, pre, ok := strings.Cut(version, "-"); ok {
		for _, id := range strings.Split(pre, ".") {
			if len(id) > 1 && id[0] == '0' && numeric.MatchString(id) {
				return fmt.Errorf("invalid version %q: numeric prerelease identifiers cannot have leading zeroes", version)
			}
		}
	}
	return nil
}

func run(args []string, out io.Writer) error {
	data, err := os.ReadFile("mygo.json")
	if err != nil {
		return err
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	var version string
	if err := json.Unmarshal(config["version"], &version); err != nil {
		return fmt.Errorf("read mygo.json version: %w", err)
	}
	if err := validate(version); err != nil {
		return err
	}
	if len(args) == 0 {
		_, err := fmt.Fprintln(out, version)
		return err
	}
	if len(args) == 1 && args[0] == "ci" {
		return ci(version, out)
	}
	if len(args) != 2 || (args[0] != "set" && args[0] != "bump") {
		return fmt.Errorf("usage: go run ./cmd/version [bump patch|minor|major | set VERSION | ci]")
	}
	if args[0] == "set" {
		version = args[1]
	} else {
		core, _, _ := strings.Cut(version, "-")
		parts := strings.Split(core, ".")
		index := -1
		switch args[1] {
		case "major":
			index = 0
		case "minor":
			index = 1
		case "patch":
			index = 2
		default:
			return fmt.Errorf("unknown bump %q: use patch, minor or major", args[1])
		}
		n, err := strconv.ParseUint(parts[index], 10, 64)
		if err != nil || n == ^uint64(0) {
			return fmt.Errorf("version component %q is too large to bump", parts[index])
		}
		parts[index] = strconv.FormatUint(n+1, 10)
		for i := index + 1; i < len(parts); i++ {
			parts[i] = "0"
		}
		version = strings.Join(parts, ".")
	}
	if err := validate(version); err != nil {
		return err
	}
	config["version"], _ = json.Marshal(version)
	data, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile("mygo.json", append(data, '\n'), 0644); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, version)
	return err
}

func ci(version string, out io.Writer) error {
	switch os.Getenv("GITHUB_REF_TYPE") {
	case "tag":
		if tag := os.Getenv("GITHUB_REF_NAME"); tag != "v"+version {
			return fmt.Errorf("release tag must be v%s, matching mygo.json (got %q)", version, tag)
		}
	case "branch":
		run, sha := os.Getenv("GITHUB_RUN_NUMBER"), os.Getenv("GITHUB_SHA")
		if !runPattern.MatchString(run) || !shaPattern.MatchString(sha) {
			return fmt.Errorf("branch builds need a positive GITHUB_RUN_NUMBER and a 40-character GITHUB_SHA")
		}
		separator := "-"
		if strings.Contains(version, "-") {
			separator = "."
		}
		version += separator + "dev." + run + ".g" + sha[:7]
	default:
		return fmt.Errorf("ci requires GITHUB_REF_TYPE to be branch or tag")
	}
	metadata := fmt.Sprintf("version=%s\ntag=v%s\nprerelease=%t\n", version, version, strings.Contains(version, "-"))
	if path := os.Getenv("GITHUB_OUTPUT"); path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		_, writeErr := io.WriteString(f, metadata)
		closeErr := f.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	_, err := io.WriteString(out, metadata)
	return err
}
