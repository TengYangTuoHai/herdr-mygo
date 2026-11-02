package herdr

import (
	"encoding/json"
	"testing"
)

// A terminal record as the core sends it: `status` is an object, not a word.
const reportedRecord = `{
	"address": "t10", "argv": ["sh"], "cols": 200, "rows": 120, "cwd": "/tmp",
	"state": "live", "status": {"state": "working"},
	"status_ms": 1793625900556, "reported": true, "turn": 0,
	"labels": {"herdr.space": "sp1", "herdr.tab": "tb1"}
}`

func TestTerminalReadsTheStatusObject(t *testing.T) {
	var term Terminal
	if err := json.Unmarshal([]byte(reportedRecord), &term); err != nil {
		t.Fatalf("a reported terminal must load: %v", err)
	}
	if got := term.Word(); got != "working" {
		t.Fatalf("word: %q", got)
	}
	if term.Status.State != "working" {
		t.Fatalf("state: %q", term.Status.State)
	}
}

// The record a live Claude Code terminal sends (terminal.get): the shell's
// argv, the agent's own app in the status, its binary named after its version.
const claudeRecord = `{
	"address": "t6", "argv": ["/bin/zsh", "-l"], "cols": 163, "rows": 43,
	"cwd": "/Users/me/project", "state": "live", "title": "Claude Code",
	"foreground": {"argv": ["claude", "--dangerously-skip-permissions"],
		"cwd": "/Users/me/project", "name": "2.1.314", "pid": 52468},
	"reported": true, "status": {"app": "claude-code", "state": "idle"},
	"status_ms": 1793627903404, "turn": 0,
	"labels": {"herdr.space": "sp1", "herdr.tab": "tb1"}
}`

func TestAgentInAShellReadsItsOwnReport(t *testing.T) {
	var term Terminal
	if err := json.Unmarshal([]byte(claudeRecord), &term); err != nil {
		t.Fatal(err)
	}
	if got := term.Status.app(); got != "claude-code" {
		t.Fatalf("app: %q", got)
	}
	p := Program(&term)
	if p.Key != "claude" || p.Icon != "claude" || !p.Agent {
		t.Fatalf("program: %+v", p)
	}
}

// sessionWithOnePane is a session holding one tab, one pane and one terminal.
func sessionWithOnePane(record string) *Session {
	var term Terminal
	if err := json.Unmarshal([]byte(record), &term); err != nil {
		panic(err)
	}
	pane := &Node{ID: "p1", Type: "block", Content: &NodeContent{Terminal: term.Address}}
	tab := &Tab{ID: "tb1", Root: &Node{ID: "tb1", Type: "row", Children: []*Node{pane}}}
	sp := &Space{ID: "sp1", Tabs: []*Tab{tab}}
	return &Session{
		spaces:    []*Space{sp},
		terminals: map[string]*Terminal{term.Address: &term},
		order:     []string{term.Address},
		homes:     map[string]string{},
		tabs:      map[string]*tabView{"tb1": {focused: "p1"}},
		query:     map[string]string{},
		seen:      map[string]bool{},
	}
}

func TestAgentsListShowsAReportedPane(t *testing.T) {
	s := sessionWithOnePane(reportedRecord)
	s.mu.Lock()
	list := s.agentsLocked()
	s.mu.Unlock()
	if len(list) != 1 {
		t.Fatalf("agents: got %d, want 1", len(list))
	}
	ag := list[0]
	if ag.Status != "working" || ag.Pane != "p1" || ag.Address != "t10" || !ag.Focused {
		t.Fatalf("agent row: %+v", ag)
	}
}

// The status event carries the object under "status" and `null` clears it.
func TestStatusEventTakesTheObject(t *testing.T) {
	s := sessionWithOnePane(reportedRecord)
	s.applyEvent("status", "t10", json.RawMessage(`{"records":{},"reported":true,"status":{"state":"blocked"}}`))
	if got := s.terminals["t10"].Word(); got != "blocked" {
		t.Fatalf("after the event: %q", got)
	}
	if got := s.terminals["t10"].Status; got == nil || got.State != "blocked" {
		t.Fatalf("status: %+v", got)
	}
	s.applyEvent("status", "t10", json.RawMessage(`{"reported":true,"status":null}`))
	if got := s.terminals["t10"].Word(); got != "" {
		t.Fatalf("after the clear: %q", got)
	}
}

// A terminal that reports a word the app does not know still shows, as the
// program's own word.
func TestAgentsListKeepsProgramWords(t *testing.T) {
	s := sessionWithOnePane(`{"address":"t10","state":"live","labels":{},"status":{"state":"thinking"},"turn":1}`)
	s.mu.Lock()
	list := s.agentsLocked()
	s.mu.Unlock()
	if len(list) != 1 || list[0].Status != "thinking" {
		t.Fatalf("agents: %+v", list)
	}
}
