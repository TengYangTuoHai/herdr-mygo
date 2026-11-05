package gitcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("HERDR_TEST_GIT_CONSOLE") == "1" && len(os.Args) == 5 && os.Args[1] == "-C" && os.Args[3] == "rev-parse" && os.Args[4] == "--show-toplevel" {
		window, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow").Call()
		fmt.Fprintf(os.Stdout, "%d\n", window)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// Exercise the same launch path as Git, using a child that reports whether
// Windows gave it a console. Removing CREATE_NO_WINDOW must fail this check.
func TestOutputWithoutConsole(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "git.exe"), data, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HERDR_TEST_GIT_CONSOLE", "1")
	out, err := Output(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatal(err)
	}
	window, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		t.Fatalf("console window output %q: %v", out, err)
	}
	if window != 0 {
		t.Fatalf("git has a console window: %#x", window)
	}
}
