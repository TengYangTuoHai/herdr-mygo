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

func TestAgentsListRestoresSeenDoneFromTerminalLabel(t *testing.T) {
	s := sessionWithOnePane(`{"address":"t10","state":"live","labels":{"herdr.seen":"3"},"status":{"state":"done"},"turn":3}`)
	s.mu.Lock()
	list := s.agentsLocked()
	s.mu.Unlock()
	if len(list) != 1 || list[0].Status != "idle" {
		t.Fatalf("restored seen agent: %+v, want one idle agent", list)
	}
}

func TestAgentsListDoesNotRestoreSeenFromEarlierTurn(t *testing.T) {
	s := sessionWithOnePane(`{"address":"t10","state":"live","labels":{"herdr.seen":"2"},"status":{"state":"done"},"turn":3}`)
	s.mu.Lock()
	list := s.agentsLocked()
	s.mu.Unlock()
	if len(list) != 1 || list[0].Status != "done" {
		t.Fatalf("new turn status: %+v, want one done agent", list)
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

// Two agents in one space, each in its own tab: only the shown tab's pane is
// the focused agent, and focusing the other one shows its tab.
func TestAgentsListFocusesOneTabOfASpace(t *testing.T) {
	var one, two Terminal
	for _, rec := range []string{
		`{"address":"t10","state":"live","labels":{},"status":{"state":"working"},"turn":0}`,
		`{"address":"t11","state":"live","labels":{},"status":{"state":"working"},"turn":0}`,
	} {
		var term Terminal
		if err := json.Unmarshal([]byte(rec), &term); err != nil {
			t.Fatal(err)
		}
		if term.Address == "t10" {
			one = term
		} else {
			two = term
		}
	}
	p1 := &Node{ID: "p1", Type: "block", Content: &NodeContent{Terminal: one.Address}}
	p2 := &Node{ID: "p2", Type: "block", Content: &NodeContent{Terminal: two.Address}}
	tb1 := &Tab{ID: "tb1", Root: &Node{ID: "tb1", Type: "row", Children: []*Node{p1}}}
	tb2 := &Tab{ID: "tb2", Root: &Node{ID: "tb2", Type: "row", Children: []*Node{p2}}}
	s := &Session{
		spaces:    []*Space{{ID: "sp1", Tabs: []*Tab{tb1, tb2}}},
		terminals: map[string]*Terminal{"t10": &one, "t11": &two},
		order:     []string{"t10", "t11"},
		homes:     map[string]string{},
		tabs:      map[string]*tabView{"tb1": {focused: "p1"}, "tb2": {focused: "p2"}},
		activeTab: map[string]string{"sp1": "tb1"},
		query:     map[string]string{},
		seen:      map[string]bool{},
	}
	s.mu.Lock()
	list := s.agentsLocked()
	s.mu.Unlock()
	if len(list) != 2 {
		t.Fatalf("agents: got %d, want 2", len(list))
	}
	for _, ag := range list {
		if want := ag.Pane == "p1"; ag.Focused != want {
			t.Fatalf("agent %s focus: got %v, want %v", ag.Pane, ag.Focused, want)
		}
	}
	if !s.FocusTerminal("t11") {
		t.Fatal("t11 must focus")
	}
	s.mu.Lock()
	list = s.agentsLocked()
	s.mu.Unlock()
	for _, ag := range list {
		if want := ag.Pane == "p2"; ag.Focused != want {
			t.Fatalf("after focusing t11, agent %s focus: got %v, want %v", ag.Pane, ag.Focused, want)
		}
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
func TestFinishedTerminalIsRemovedFromAgentsAfterSnapshot(t *testing.T) {
	s := sessionWithOnePane(reportedRecord)
	s.activeTab = map[string]string{"sp1": "tb1"}
	s.applySnapshot(json.RawMessage(`{"terminals":[{"address":"t10","state":"finished","status":{"state":"working"}}]}`))
	s.mu.Lock()
	list := s.agentsLocked()
	s.mu.Unlock()
	if len(list) != 0 {
		t.Fatalf("finished terminal still listed as agent: %+v", list)
	}
}

func TestHandleLineAppliesStatusEvent(t *testing.T) {
	s := sessionWithOnePane(reportedRecord)
	s.activeTab = map[string]string{"sp1": "tb1"}
	s.git = map[string]gitInfo{}
	s.gitBusy = map[string]bool{}
	s.handleLine([]byte(`{"method":"event","params":{"type":"status","terminal":"t10","data":{"status":{"state":"done"}}}}`))
	if got := s.terminals["t10"].Word(); got != "done" {
		t.Fatalf("status after final event: %q", got)
	}
}

func TestAgentsListKeepsProgramWords(t *testing.T) {
	s := sessionWithOnePane(`{"address":"t10","state":"live","labels":{},"status":{"state":"thinking"},"turn":1}`)
	s.mu.Lock()
	list := s.agentsLocked()
	s.mu.Unlock()
	if len(list) != 1 || list[0].Status != "thinking" {
		t.Fatalf("agents: %+v", list)
	}
}

// An agent on a remote machine's space carries that machine's name; an agent
// on this computer carries none.
func TestAgentsListCarriesTheMachineOfARemoteSpace(t *testing.T) {
	remote := "gpu"
	s := sessionWithOnePane(reportedRecord)
	var term Terminal
	if err := json.Unmarshal([]byte(`{"address":"t11","state":"live","labels":{},"status":{"state":"idle"},"turn":0}`), &term); err != nil {
		t.Fatal(err)
	}
	p2 := &Node{ID: "p2", Type: "block", Content: &NodeContent{Terminal: "t11"}}
	tb2 := &Tab{ID: "gpu/tb2", Root: &Node{ID: "gpu/tb2", Type: "row", Children: []*Node{p2}}}
	s.spaces = append(s.spaces, &Space{ID: "gpu/sp2", Machine: &remote, Tabs: []*Tab{tb2}})
	s.terminals["t11"] = &term
	s.mu.Lock()
	list := s.agentsLocked()
	s.mu.Unlock()
	if len(list) != 2 {
		t.Fatalf("agents: got %d, want 2", len(list))
	}
	for _, ag := range list {
		want := ""
		if ag.Space == "gpu/sp2" {
			want = "gpu"
		}
		if ag.Machine != want {
			t.Fatalf("agent %s of %q: machine %q, want %q", ag.Pane, ag.Space, ag.Machine, want)
		}
	}
}
