package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("HERDR_TEST_HC_CONNECT") == "1" && len(os.Args) == 2 && os.Args[1] == "connect" {
		// Bound a failed handshake so Connect cannot hang the test indefinitely.
		time.AfterFunc(5*time.Second, func() { os.Exit(2) })
		line, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
		if err != nil {
			os.Exit(2)
		}
		var hello struct {
			Method string `json:"method"`
			Params struct {
				Framed bool `json:"framed"`
			} `json:"params"`
		}
		if json.Unmarshal(line, &hello) != nil || hello.Method != "hello" || !hello.Params.Framed {
			os.Exit(2)
		}
		window, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow").Call()
		fmt.Fprintf(os.Stdout, "{\"jsonrpc\":\"2.0\",\"id\":0,\"result\":{\"console_window\":%d}}\n", window)
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// Removing the Windows process options must expose a console in this child,
// while the real Connect path must still exchange its hello over stdin/stdout.
func TestConnectWithoutConsole(t *testing.T) {
	t.Setenv("HERDR_TEST_HC_CONNECT", "1")
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	client, err := Connect(binary)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.Close)
	var hello struct {
		ConsoleWindow uintptr `json:"console_window"`
	}
	if err := json.Unmarshal(client.Hello, &hello); err != nil {
		t.Fatal(err)
	}
	if hello.ConsoleWindow != 0 {
		t.Fatalf("hc connect has a console window: %#x", hello.ConsoleWindow)
	}
}
