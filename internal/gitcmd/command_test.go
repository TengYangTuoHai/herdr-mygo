package gitcmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOutput(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "repo with spaces")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", "--quiet", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := Output(dir, "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != "?? new.txt" {
		t.Fatalf("git status = %q, want %q", got, "?? new.txt")
	}
}

func TestOutputExitError(t *testing.T) {
	_, err := Output(t.TempDir(), "rev-parse", "--show-toplevel")
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() == 0 {
		t.Fatalf("git outside a repository: want nonzero exit error, got %v", err)
	}
}
