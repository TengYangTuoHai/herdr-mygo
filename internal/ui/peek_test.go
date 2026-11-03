package ui

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/plugins/terminal"
	myui "github.com/egoist/mygo/ui"

	"herdr-mygo/internal/herdr"
)

// peekWindow draws a peek of addr the way the window draws its overlays.
func peekWindow(t *testing.T, app *App) *myui.Tester {
	t.Helper()
	app.W, app.H = 1280, 820
	tt := myui.NewTester(func(c *myui.Context) { app.overlays(c) }, 1280, 820)
	tt.Frame()
	return tt
}

// TestPeekBarOffersKeepAndClose draws the bar over a peeked sub-agent: it says
// which terminal is being looked at, and offers the keep that gives it a pane,
// with the close that puts the look away.
func TestPeekBarOffersKeepAndClose(t *testing.T) {
	RegisterFonts()
	app := NewApp(&herdr.Session{}, ThemeByName("graphite"))
	app.overlay = "peek"
	app.peek = peekState{addr: "t2"}
	tt := peekWindow(t, app)
	if !tt.HasText("t2") {
		t.Fatalf("the bar does not name the terminal it looks at: %q", tt.Texts())
	}
	for _, label := range []string{"↗ keep", "✕"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("the bar has no %q: %q", label, tt.Texts())
		}
	}

	// The keep asks the session for a pane. This session holds no terminal to
	// give one to, so the window says why instead of doing nothing.
	keep, _ := tt.Find("↗ keep")
	tt.ClickAt(keep.X+keep.W/2, keep.Y+keep.H/2)
	deadline := time.Now().Add(2 * time.Second)
	for {
		app.mu.Lock()
		notes := append([]Toast(nil), app.notes...)
		app.mu.Unlock()
		if len(notes) > 0 {
			if notes[0].Title != "Cannot keep the sub-agent" {
				t.Fatalf("the keep said %q: %s", notes[0].Title, notes[0].Body)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the bar's keep did nothing")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestPeekClosesWhenTheKeepAnswers keeps the look open until the keep it asked
// for comes back: a peek that closed on the click would leave a click with no
// word of what became of it.
func TestPeekClosesWhenTheKeepAnswers(t *testing.T) {
	RegisterFonts()
	app := NewApp(&herdr.Session{}, ThemeByName("graphite"))
	app.overlay = "peek"
	app.peek = peekState{addr: "t2"}
	peekWindow(t, app)
	if app.overlay != "peek" {
		t.Fatal("the look closed before its keep answered")
	}

	app.mu.Lock()
	app.peek.kept = true
	app.mu.Unlock()
	peekWindow(t, app)
	if app.overlay == "peek" {
		t.Fatal("the look stayed open over the pane its keep made")
	}
}

// TestClosingAPeekDropsItsAttachment ends the look, not the terminal: the
// attachment it made is forgotten, so a later frame cannot draw a view of a
// terminal the window no longer looks at.
func TestClosingAPeekDropsItsAttachment(t *testing.T) {
	app := NewApp(&herdr.Session{}, ThemeByName("graphite"))
	app.overlay = "peek"
	app.peek = peekState{addr: "t2"}
	app.CloseOverlay()
	if app.peek.addr != "" {
		t.Errorf("the closed look still holds %q", app.peek.addr)
	}
}

// TestPeekNamesATerminalTheWayTheListDoes keeps the bar's name for a terminal
// the same as the row's under it: its name, else its template, else its
// program, else its address.
func TestPeekNamesATerminalTheWayTheListDoes(t *testing.T) {
	named := worker("t2", "t1", "worker one")
	if got := terminalName(named, "t2"); got != "worker one" {
		t.Errorf("a named terminal is called %q, want its name", got)
	}
	templated := &herdr.Terminal{Address: "t3", Template: strp("pi")}
	if got := terminalName(templated, "t3"); got != "pi" {
		t.Errorf("a terminal with a template is called %q, want its template", got)
	}
	bare := &herdr.Terminal{Address: "t4"}
	if got, want := terminalName(bare, "t4"), herdr.Program(bare).Name; got != want {
		t.Errorf("a terminal with nothing else is called %q, want its program %q", got, want)
	}
	if got := terminalName(nil, "t5"); got != "t5" {
		t.Errorf("a terminal the session does not know is called %q, want its address", got)
	}
}

// peekConn is a terminal's other end for a peek drawn without a core: what is
// typed arrives in written, and the view's reads wait until it closes.
type peekConn struct {
	mu      sync.Mutex
	written []byte
	closed  chan struct{}
	once    sync.Once
}

func newPeekConn() *peekConn { return &peekConn{closed: make(chan struct{})} }

func (c *peekConn) Read(p []byte) (int, error) { <-c.closed; return 0, io.EOF }

func (c *peekConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	c.written = append(c.written, p...)
	c.mu.Unlock()
	return len(p), nil
}

func (c *peekConn) Close() error {
	c.once.Do(func() { close(c.closed) })
	return nil
}

func (c *peekConn) typed() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return string(c.written)
}

// TestPeekTakesTheKeysOfTheTerminalItShows types into a peek: the look is a
// terminal, not a picture of one, so what is typed goes to the worker.
func TestPeekTakesTheKeysOfTheTerminalItShows(t *testing.T) {
	RegisterFonts()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	conn := newPeekConn()
	term, err := terminal.New(terminal.Options{
		Conn:  conn,
		Font:  terminal.Font{Family: TerminalFont, Size: 14},
		Theme: terminalTheme(ThemeByName("graphite")),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer term.Close()

	app := NewApp(&herdr.Session{}, ThemeByName("graphite"))
	app.overlay = "peek"
	app.peek = peekState{addr: "t2", term: term}
	tt := myui.NewTester(app.View, 1280, 820)
	tt.SetFocused(true)
	tt.Frame()
	tt.Type("hi")
	tt.Frame()
	if got := conn.typed(); !strings.Contains(got, "hi") {
		t.Fatalf("the peek took %q of what was typed, want hi", got)
	}
}
