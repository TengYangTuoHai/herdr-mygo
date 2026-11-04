package herdr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"herdr-mygo/internal/core"
)

// tabView is what each screen keeps of a tab: its focus, zoom and history.
type tabView struct {
	focused string
	before  []string
	zoomed  string
}

// Session is the window's connection to the core and its model.
type Session struct {
	Core     *core.Client
	OnChange func() // called (on any goroutine) when something changed

	mu            sync.Mutex
	spaces        []*Space
	docs          map[string]*docSlot
	versions      map[string]uint64
	machines      []*Machine
	machineByName map[string]*Machine
	machineOrder  []string // the order the person dragged the groups into
	machineSig    string
	homes         map[string]string
	wantActive    string
	terminals     map[string]*Terminal
	order         []string
	active        string
	activeTab     map[string]string
	tabs          map[string]*tabView
	query         map[string]string // pane -> terminal shown in its place
	borrowed      map[string]string // pane -> the terminal the pane really runs
	git           map[string]gitInfo
	gitBusy       map[string]bool
	agents        []*Agent
	agentSeq      int
	seen          map[string]bool
	subCh         uint32
	loadErr       error
	savedAt       time.Time
}

// Agent is one terminal whose program reports what it is doing.
type Agent struct {
	Address   string  `json:"address"`
	Pane      string  `json:"pane"`
	Space     string  `json:"space"`
	SpaceName string  `json:"space_name"`
	Machine   string  `json:"machine,omitempty"` // "" is this computer
	Tab       string  `json:"tab"`
	TabIndex  int     `json:"tab_index"`
	TabName   string  `json:"tab_name"`
	Status    string  `json:"status"`
	App       *string `json:"app"`
	Title     string  `json:"title"`
	Seq       int     `json:"seq"`
	Pinned    bool    `json:"pinned"`
	Focused   bool    `json:"focused"`
	Parent    *string `json:"parent"`
	Program   ProgramInfo
}

// PaneInfo is one pane of the shown tab, as the pane header and tab bar read it.
type PaneInfo struct {
	Pane     string
	Terminal string
	Mod      string
	Tab      string
	Title    string
	Cwd      string
	Branch   string
	Status   string
	App      string
	Program  ProgramInfo
	Focused  bool
	Zoomed   bool
	Movable  bool
}

// docSlot is how far this window got with one machine's document.
type docSlot struct {
	loaded    bool
	available bool
	err       string
	spaces    int
}

// gitInfo is what the sidebar and status bar say about a folder.
type gitInfo struct {
	Repo    string
	Branch  string
	Ahead   int
	Behind  int
	Checked time.Time
}

// Connect reads the world from the user's core and starts following it.
func Connect(bin string, onChange func()) (*Session, error) {
	cl, err := core.Connect(bin)
	if err != nil {
		return nil, err
	}
	s := &Session{
		Core:          cl,
		OnChange:      onChange,
		terminals:     map[string]*Terminal{},
		docs:          map[string]*docSlot{},
		homes:         map[string]string{},
		versions:      map[string]uint64{},
		machineByName: map[string]*Machine{},
		activeTab:     map[string]string{},
		tabs:          map[string]*tabView{},
		query:         map[string]string{},
		borrowed:      map[string]string{},
		git:           map[string]gitInfo{},
		gitBusy:       map[string]bool{},
		seen:          map[string]bool{},
	}
	if err := s.reload(); err != nil {
		return nil, err
	}
	go s.follow()
	go s.pollForever()
	return s, nil
}

// reload reads the document and the terminals, then subscribes. The event
// snapshot is applied before the document, so a terminal that arrived in
// between still meets its space.
func (s *Session) reload() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Documents first: a fresh snapshot replaces terminals wholesale.
	s.readDoc("")
	raw, err := s.Core.Call(ctx, "machine.list", map[string]any{})
	if err == nil {
		s.adoptMachines(raw, true)
	} else if s.loadErr == nil {
		s.loadErr = err
	}
	var res struct {
		Terminals []json.RawMessage `json:"terminals"`
	}
	if raw, err := s.Core.Call(ctx, "terminal.list", map[string]any{"all": true}); err == nil {
		_ = json.Unmarshal(raw, &res)
	}
	if ch, snap, err := s.Core.Subscribe(ctx); err == nil {
		s.subCh = ch
		s.applySnapshot(snap)
	}
	s.mu.Lock()
	orderSpaces(s.spaces)
	s.mu.Unlock()
	s.reindex()
	s.LoadView()
	return nil
}

// readDoc reads one machine's document ("" is this computer).
func (s *Session) readDoc(machine string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	params := map[string]any{"name": "herdr-app"}
	if machine != "" {
		params["on"] = machine
	}
	raw, err := s.Core.Call(ctx, "doc.get", params)
	if err != nil {
		s.mu.Lock()
		slot := s.docs[machine]
		if slot == nil {
			slot = &docSlot{}
			s.docs[machine] = slot
		}
		slot.err = err.Error()
		s.mu.Unlock()
		if machine == "" && s.loadErr == nil {
			s.loadErr = err
		}
		return
	}
	s.applyDoc(machine, raw)
}

// applyDoc replaces one machine's spaces with what its document says.
func (s *Session) applyDoc(machine string, raw json.RawMessage) {
	var res struct {
		Value   json.RawMessage `json:"value"`
		Version uint64          `json:"version"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.docs == nil {
		s.docs = map[string]*docSlot{}
	}
	if s.versions == nil {
		s.versions = map[string]uint64{}
	}
	s.versions[machine] = res.Version
	var d docFile
	if len(res.Value) == 0 || string(res.Value) == "null" {
		s.replaceMachine(machine, nil)
		s.docs[machine] = &docSlot{loaded: true, available: true}
		return
	}
	if json.Unmarshal(res.Value, &d) != nil {
		s.docs[machine] = &docSlot{loaded: true, available: false, err: "unreadable document"}
		return
	}
	var tag *string
	if machine != "" {
		m := machine
		tag = &m
	}
	spaces := fromDoc(d, tag)
	s.replaceMachine(machine, spaces)
	s.docs[machine] = &docSlot{loaded: true, available: true, spaces: len(spaces)}
}

// replaceMachine swaps one machine's spaces into the model, in its place.
// The active space keeps its place when the same document still has it.
func (s *Session) replaceMachine(machine string, spaces []*Space) {
	was := s.active
	var at = -1
	var keep []*Space
	for _, sp := range s.spaces {
		mine := sameMachine(sp.Machine, machineTag(machine))
		if mine && at < 0 {
			at = len(keep)
		}
		if mine {
			for _, t := range sp.Tabs {
				if _, ok := s.tabs[t.ID]; !ok {
					delete(s.tabs, t.ID)
				}
			}
			continue
		}
		keep = append(keep, sp)
	}
	if at < 0 {
		at = len(keep)
	}
	out := append([]*Space{}, keep[:at]...)
	out = append(out, spaces...)
	out = append(out, keep[at:]...)
	s.spaces = out
	if was != "" {
		s.active = ""
		for _, sp := range out {
			if sp.ID == was {
				s.active = was
			}
		}
	}
	// The space this screen last showed may only arrive with its machine's
	// document; take it when it does.
	if s.active == "" && s.wantActive != "" {
		for _, sp := range out {
			if sp.ID == s.wantActive {
				s.active = s.wantActive
			}
		}
	}
	if s.active != "" && s.active == s.wantActive {
		s.wantActive = ""
	}
	orderSpaces(s.spaces)
}

// machineTag turns "" into the nil the model uses for this computer.
func machineTag(name string) *string {
	if name == "" {
		return nil
	}
	return &name
}

// applySnapshot takes the terminals an events.subscribe reply carries.
func (s *Session) applySnapshot(raw json.RawMessage) {
	var res struct {
		Terminals []*Terminal `json:"terminals"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.terminals = map[string]*Terminal{}
	s.order = nil
	for _, t := range res.Terminals {
		s.terminals[t.Address] = t
		s.order = append(s.order, t.Address)
	}
}

func (s *Session) refreshTerminals() {
	if s.Core == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	raw, err := s.Core.Call(ctx, "terminal.list", map[string]any{"all": true})
	if err != nil {
		return
	}
	s.applySnapshot(raw)
	s.changed()
}

// follow reads the event stream until the core goes.
func (s *Session) follow() {
	for {
		ch, err := s.Core.Stream(s.subCh)
		if err != nil {
			if !s.resubscribe() {
				return
			}
			continue
		}
		buf := []byte{}
		for {
			item, ok := <-ch
			if !ok {
				if !s.resubscribe() {
					return
				}
				break
			}
			if item.Kind == core.FrameClose {
				if item.Data == nil {
					return // the connection to the core went
				}
				if len(buf) > 0 {
					s.handleLine(buf)
				}
				if !s.resubscribe() {
					return
				}
				break
			}
			buf = append(buf, item.Data...)
			for {
				i := indexByte(buf, '\n')
				if i < 0 {
					break
				}
				line := buf[:i]
				buf = buf[i+1:]
				s.handleLine(line)
			}
		}
	}
}

func (s *Session) resubscribe() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	channel, snapshot, err := s.Core.Subscribe(ctx)
	if err != nil {
		return false
	}
	s.subCh = channel
	s.applySnapshot(snapshot)
	s.changed()
	return true
}

func indexByte(b []byte, c byte) int {
	for i := range b {
		if b[i] == c {
			return i
		}
	}
	return -1
}

// handleLine applies one event notification.
func (s *Session) handleLine(line []byte) {
	var msg struct {
		Method string `json:"method"`
		Params struct {
			Seq       uint64          `json:"seq"`
			Type      string          `json:"type"`
			Terminal  string          `json:"terminal"`
			Data      json.RawMessage `json:"data"`
			Terminals json.RawMessage `json:"terminals"`
			Machine   string          `json:"machine"`
		} `json:"params"`
	}
	if err := json.Unmarshal(line, &msg); err != nil {
		return
	}
	switch msg.Method {
	case "heartbeat":
		return
	case "missed":
		s.applySnapshot(msg.Params.Terminals)
		s.refreshMachines()
		s.readDoc("")
		for _, m := range s.Machines() {
			if m.Connected() {
				s.readDoc(m.Name)
			}
		}
		s.changed()
		return
	case "event":
		if msg.Params.Type == "doc" {
			s.rereadDoc(msg.Params.Machine)
			s.changed()
			return
		}
		if s.applyEvent(msg.Params.Type, msg.Params.Terminal, msg.Params.Data) {
			s.changed()
		}
	}
}

// rereadDoc reads a document again after the core says it changed.
func (s *Session) rereadDoc(machine string) { s.readDoc(machine) }

// applyEvent changes the terminal the event names. It reports a change.
func (s *Session) applyEvent(kind, address string, data json.RawMessage) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.terminals[address]
	if kind == "created" || kind == "restarted" {
		var rec Terminal
		if json.Unmarshal(data, &rec) != nil {
			return false
		}
		if t == nil {
			s.order = append(s.order, address)
		}
		s.terminals[address] = &rec
		return true
	}
	if t == nil {
		return false
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(data, &fields)
	switch kind {
	case "removed":
		delete(s.terminals, address)
		for i, a := range s.order {
			if a == address {
				s.order = append(s.order[:i], s.order[i+1:]...)
				break
			}
		}
		return true
	case "finished", "lost":
		if kind == "finished" {
			t.State = "finished"
		}
		if f := fields["reason"]; f != nil {
			var reason *string
			_ = json.Unmarshal(f, &reason)
			t.Reason = reason
		}
		if f := fields["exit_code"]; f != nil {
			var code *int
			_ = json.Unmarshal(f, &code)
			t.ExitCode = code
		}
		if kind == "lost" {
			t.Pid = nil
		}
		return true
	case "status":
		if f := fields["status"]; f != nil {
			if string(f) == "null" {
				t.Status = nil
			} else {
				var st Status
				if json.Unmarshal(f, &st) == nil {
					t.Status = &st
				}
			}
		}
		if f := fields["reported"]; f != nil {
			_ = json.Unmarshal(f, &t.Reported)
		}
		return true
	case "turn":
		if f := fields["turn"]; f != nil {
			_ = json.Unmarshal(f, &t.Turn)
		}
		return true
	case "title":
		if f := fields["title"]; f != nil {
			_ = json.Unmarshal(f, &t.Title)
		}
		return true
	case "labels":
		if f := fields["labels"]; f != nil {
			_ = json.Unmarshal(f, &t.Labels)
		}
		if len(data) > 0 {
			var rec struct {
				Record *Terminal `json:"record"`
			}
			if json.Unmarshal(data, &rec) == nil && rec.Record != nil {
				s.terminals[rec.Record.Address] = rec.Record
			}
		}
		return true
	case "keep":
		if f := fields["keep"]; f != nil {
			_ = json.Unmarshal(f, &t.Keep)
		}
		return true
	case "size":
		if f := fields["cols"]; f != nil {
			_ = json.Unmarshal(f, &t.Cols)
		}
		if f := fields["rows"]; f != nil {
			_ = json.Unmarshal(f, &t.Rows)
		}
		return true
	case "foreground":
		if string(data) == "null" {
			t.Foreground = nil
		} else {
			var fg Foreground
			if json.Unmarshal(data, &fg) == nil {
				t.Foreground = &fg
			}
		}
		return true
	}
	return false
}

// changed tells the window to draw again.
func (s *Session) changed() {
	s.reindex()
	if time.Since(s.savedAt) > time.Second {
		s.savedAt = time.Now()
		s.SaveView()
	}
	if s.OnChange != nil {
		s.OnChange()
	}
}

// reindex rebuilds what the views read: agent rows and the git of places.
func (s *Session) reindex() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents = s.agentsLocked()
	if s.active == "" && len(s.spaces) > 0 {
		s.active = s.spaces[0].ID
	}
	s.ensureFocusLocked()
	for _, sp := range s.spaces {
		for _, t := range sp.Tabs {
			for _, b := range t.Root.blocks() {
				if addr := b.terminal(); addr != "" {
					s.wantGitLocked(s.terminals[addr], sp)
				}
			}
		}
	}
}

// ensureFocusLocked gives tabs without a focus the first pane.
func (s *Session) ensureFocusLocked() {
	for _, sp := range s.spaces {
		for _, t := range sp.Tabs {
			v := s.tabs[t.ID]
			if v == nil {
				v = &tabView{}
				s.tabs[t.ID] = v
			}
			if s.activeTab[sp.ID] == "" || t.ID == s.activeTab[sp.ID] {
				s.activeTab[sp.ID] = t.ID
			}
			if v.focused == "" || t.Root.find(v.focused) == nil {
				if blocks := t.Root.blocks(); len(blocks) > 0 {
					v.focused = blocks[0].ID
				}
			}
			if v.zoomed != "" && t.Root.find(v.zoomed) == nil {
				v.zoomed = ""
			}
		}
	}
}

// agentsLocked lists live terminals that report a status, in space, tab,
// pane order.
func (s *Session) agentsLocked() []*Agent {
	// Only the shown tab owns the keyboard: every other tab keeps a focus of
	// its own, so marking those would light up one agent per tab.
	shownSpace, shownTab := s.activeLocked()
	shownTabID := ""
	if shownTab != nil {
		shownTabID = shownTab.ID
	}
	var out []*Agent
	for _, sp := range s.spaces {
		for ti, t := range sp.Tabs {
			for _, b := range t.Root.blocks() {
				addr := b.terminal()
				term := s.terminals[addr]
				if term == nil || !term.Live() || term.Word() == "" || term.Word() == "unknown" {
					continue
				}
				app := ""
				if apps := runningApps(term); len(apps) > 0 {
					app = apps[0]
				}
				home := s.homes[SpaceMachine(sp)]
				a := &Agent{
					Program:   Program(term),
					Address:   addr,
					Pane:      b.ID,
					Space:     sp.ID,
					SpaceName: spaceNameWith(sp, s.terminals, home),
					Machine:   SpaceMachine(sp),
					Tab:       t.ID,
					TabIndex:  ti + 1,
					TabName:   TabName(t, ti+1),
					Status:    term.Word(),
					Title:     term.AgentTitle(),
					Seq:       term.Turn,
					Pinned:    term.Labels["herdr.pinned"] == "true",
					Focused: shownSpace != nil && shownSpace.ID == sp.ID && shownTabID == t.ID &&
						s.tabs[t.ID] != nil && s.tabs[t.ID].focused == b.ID,
				}
				if app != "" {
					a.App = &app
				}
				if term.Parent != nil {
					a.Parent = term.Parent
				}
				seenTurn := term.Labels["herdr.seen"] == fmt.Sprint(term.Turn)
				if (s.seen[addr] || seenTurn) && a.Status == "done" {
					a.Status = "idle"
				}
				out = append(out, a)
			}
		}
	}
	return out
}

// gitKey names a folder on a machine ("" is this computer).
func gitKey(on, dir string) string { return on + "\x00" + dir }

// machineOf is a space's machine name, "" for this computer.
func machineOf(sp *Space) string {
	if sp == nil || sp.Machine == nil {
		return ""
	}
	return *sp.Machine
}

// SpaceName names a space, "~" being the home of the machine it is on.
func (s *Session) SpaceName(sp *Space) string {
	if sp == nil {
		return ""
	}
	on := SpaceMachine(sp)
	s.mu.Lock()
	home := s.homes[on]
	s.mu.Unlock()
	if home == "" && on == "" {
		home, _ = os.UserHomeDir()
	}
	return spaceNameWith(sp, s.termSnapshot(), home)
}

// MachineHome is a machine's home folder, "" until known.
func (s *Session) MachineHome(name string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.homes[name]
}

// wantGitLocked looks up a place's repo and branch, in the background.
func (s *Session) wantGitLocked(term *Terminal, sp *Space) {
	on := machineOf(sp)
	dir := sp.Folder
	if term != nil {
		if term.Foreground != nil && term.Foreground.Cwd != "" {
			dir = term.Foreground.Cwd
		} else if term.Cwd != "" {
			dir = term.Cwd
		}
	}
	if dir == "" {
		return
	}
	key := gitKey(on, dir)
	if g, ok := s.git[key]; ok && time.Since(g.Checked) < 5*time.Second {
		return
	}
	if s.gitBusy[key] {
		return
	}
	s.gitBusy[key] = true
	go s.readGit(on, dir)
}

// readGit asks git for a folder's repo, branch and how it stands: here, or
// through the machine's core when the folder is on another computer.
func (s *Session) readGit(on, dir string) {
	info := gitInfo{Checked: time.Now()}
	branch, err := s.gitOut(on, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		info.Branch = strings.TrimSpace(branch)
		if top, err := s.gitOut(on, dir, "rev-parse", "--show-toplevel"); err == nil {
			info.Repo = filepath.Base(strings.TrimSpace(top))
		}
		if counts, err := s.gitOut(on, dir, "rev-list", "--left-right", "--count", "@{upstream}...HEAD"); err == nil {
			_, _ = fmt.Sscanf(strings.TrimSpace(counts), "%d %d", &info.Behind, &info.Ahead)
		}
	}
	key := gitKey(on, dir)
	s.mu.Lock()
	s.git[key] = info
	delete(s.gitBusy, key)
	s.mu.Unlock()
	s.changed()
}

// gitOut runs one git command for a folder, on the machine that holds it.
func (s *Session) gitOut(on, dir string, args ...string) (string, error) {
	argv := append([]string{"git", "-C", dir}, args...)
	if on == "" {
		out, err := exec.Command(argv[0], argv[1:]...).Output()
		return string(out), err
	}
	res, err := s.Exec(on, "", argv, 15*time.Second)
	if err != nil {
		return "", err
	}
	if res.Code != nil && *res.Code != 0 {
		return res.Stdout, fmt.Errorf("git exited %d", *res.Code)
	}
	return res.Stdout, nil
}

// pollForever keeps the live folders, the git of places and the machines
// current.
func (s *Session) pollForever() {
	t := time.NewTicker(3 * time.Second)
	defer t.Stop()
	machines := time.NewTicker(15 * time.Second)
	defer machines.Stop()
	for {
		select {
		case <-t.C:
			if s.Core == nil {
				return
			}
			s.refreshTerminals()
		case <-machines.C:
			if s.Core == nil {
				return
			}
			s.refreshMachines()
		}
	}
}

// Places returns the spaces and their tabs, as the views show them.
func (s *Session) Places() []*Space { return s.spacesLocked() }

func (s *Session) spacesLocked() []*Space {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Space, 0, len(s.spaces))
	for i, sp := range s.spaces {
		cp := *sp
		cp.Index = i + 1
		cp.Active = sp.ID == s.active
		cp.Tabs = append([]*Tab(nil), sp.Tabs...)
		shown := s.activeTab[sp.ID]
		for j, t := range cp.Tabs {
			tc := *t
			tc.Index = j + 1
			tc.Active = t.ID == shown
			cp.Tabs[j] = &tc
		}
		out = append(out, &cp)
	}
	return out
}

// Active returns the shown space and tab.
func (s *Session) Active() (*Space, *Tab) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeLocked()
}

func (s *Session) activeLocked() (*Space, *Tab) {
	for _, sp := range s.spaces {
		if sp.ID != s.active {
			continue
		}
		if len(sp.Tabs) == 0 {
			return sp, nil
		}
		if id := s.activeTab[sp.ID]; id != "" {
			for _, t := range sp.Tabs {
				if t.ID == id {
					return sp, t
				}
			}
		}
		return sp, sp.Tabs[0]
	}
	if len(s.spaces) > 0 {
		sp := s.spaces[0]
		if len(sp.Tabs) > 0 {
			return sp, sp.Tabs[0]
		}
		return sp, nil
	}
	return nil, nil
}

// ActiveSpaceID returns the shown space's id.
func (s *Session) ActiveSpaceID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

// Terminals returns a copy of every terminal the core runs.
func (s *Session) Terminals() []*Terminal {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Terminal, 0, len(s.order))
	for _, a := range s.order {
		if t := s.terminals[a]; t != nil {
			cp := *t
			out = append(out, &cp)
		}
	}
	return out
}

// Terminal returns one terminal record.
func (s *Session) Terminal(addr string) *Terminal {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t := s.terminals[addr]; t != nil {
		cp := *t
		return &cp
	}
	return nil
}

// Agents returns the agent rows, most urgent first.
func (s *Session) Agents() []*Agent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]*Agent(nil), s.agents...)
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := StatusRank(out[i].Status), StatusRank(out[j].Status)
		if ri != rj {
			return ri > rj
		}
		return out[i].Seq > out[j].Seq
	})
	return out
}

// Panes returns the panes of a tab.
func (s *Session) Panes(tabID string) []PaneInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []PaneInfo
	for _, sp := range s.spaces {
		for _, t := range sp.Tabs {
			if t.ID != tabID {
				continue
			}
			v := s.tabs[t.ID]
			if v == nil {
				continue
			}
			for _, b := range t.Root.blocks() {
				addr := b.terminal()
				if shown := s.query[b.ID]; shown != "" {
					addr = shown
				}
				info := PaneInfo{Pane: b.ID, Terminal: addr, Tab: t.ID}
				if b.Content != nil {
					info.Mod = b.Content.Mod
				}
				if term := s.terminals[addr]; term != nil {
					info.Program = Program(term)
					info.Status = term.Word()
					info.Title = term.Title
					if term.Foreground != nil {
						info.Cwd = term.Foreground.Cwd
						info.App = term.Foreground.Name
					}
					if info.Cwd == "" {
						info.Cwd = term.Cwd
					}
					if info.Title == "" {
						info.Title = info.Program.Name
					}
					if g, ok := s.git[gitKey(machineOf(sp), info.Cwd)]; ok {
						info.Branch = g.Branch
					}
				}
				if info.Mod != "" && info.Title == "" {
					info.Title = info.Mod
				}
				info.Focused = v.focused == b.ID
				info.Zoomed = v.zoomed != "" && v.zoomed == b.ID
				info.Movable = info.Zoomed || len(t.Root.blocks()) < 2
				out = append(out, info)
			}
		}
	}
	return out
}

// FocusedPane returns the shown tab's focused pane.
func (s *Session) FocusedPane() PaneInfo {
	_, t := s.Active()
	if t == nil {
		return PaneInfo{}
	}
	for _, p := range s.Panes(t.ID) {
		if p.Focused {
			return p
		}
	}
	return PaneInfo{}
}

// FocusedTerminal returns the terminal the keyboard is in, if any.
func (s *Session) FocusedTerminal() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, t := s.activeLocked()
	if t == nil {
		return ""
	}
	if v := s.tabs[t.ID]; v != nil && v.focused != "" {
		if b := t.Root.find(v.focused); b != nil {
			if shown := s.query[b.ID]; shown != "" {
				return shown
			}
			return b.terminal()
		}
	}
	return ""
}

// Zoomed returns the shown tab's zoomed pane, "" for none.
func (s *Session) Zoomed() string {
	_, t := s.Active()
	if t == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if v := s.tabs[t.ID]; v != nil {
		return v.zoomed
	}
	return ""
}

// Branch returns the branch of the shown space's first terminal.
func (s *Session) Branch() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sp := range s.spaces {
		if sp.ID != s.active {
			continue
		}
		for _, t := range sp.Tabs {
			for _, b := range t.Root.blocks() {
				term := s.terminals[b.terminal()]
				if term == nil {
					continue
				}
				dir := term.Cwd
				if term.Foreground != nil && term.Foreground.Cwd != "" {
					dir = term.Foreground.Cwd
				}
				if g, ok := s.git[gitKey(machineOf(sp), dir)]; ok {
					return g.Branch
				}
			}
		}
	}
	return ""
}

// Machine returns the shown space's machine, nil for this computer.
func (s *Session) Machine() *string {
	sp, _ := s.Active()
	if sp == nil {
		return nil
	}
	return sp.Machine
}

// LoadError returns the last error reading the document, if any.
func (s *Session) LoadError() error { return s.loadErr }

// spacing keeps doc writes serialized.
var writeMu sync.Mutex

// saveDoc writes every machine's share of the model, re-reading on a
// conflict. A machine that is not connected keeps its old document.
func (s *Session) saveDoc() {
	writeMu.Lock()
	defer writeMu.Unlock()
	s.mu.Lock()
	machines := map[string]*string{"": nil}
	for _, sp := range s.spaces {
		if sp.Machine != nil && *sp.Machine != "" {
			machines[*sp.Machine] = sp.Machine
		}
	}
	if s.Core == nil {
		s.mu.Unlock()
		return
	}
	type write struct {
		machine string
		value   []byte
		version uint64
	}
	var writes []write
	for name, tag := range machines {
		if name != "" {
			if m := s.machineByName[name]; m == nil || !m.Connected() {
				continue
			}
		}
		value, err := json.Marshal(captured(s.spaces, tag))
		if err != nil {
			continue
		}
		writes = append(writes, write{machine: name, value: value, version: s.versions[name]})
	}
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for i := range writes {
		w := &writes[i]
		for try := 0; try < 3; try++ {
			params := map[string]any{"name": "herdr-app", "value": json.RawMessage(w.value), "version": w.version}
			if w.machine != "" {
				params["on"] = w.machine
			}
			raw, err := s.Core.Call(ctx, "doc.set", params)
			if err == nil {
				var res struct {
					Version uint64 `json:"version"`
				}
				if json.Unmarshal(raw, &res) == nil {
					s.mu.Lock()
					s.versions[w.machine] = res.Version
					s.mu.Unlock()
				}
				break
			}
			var rpcErr *core.Error
			if !errors.As(err, &rpcErr) || rpcErr.Kind() != "conflict" {
				if w.machine == "" {
					s.mu.Lock()
					s.loadErr = err
					s.mu.Unlock()
				}
				break
			}
			// Someone else wrote first: take their version, write ours on top.
			get := map[string]any{"name": "herdr-app"}
			if w.machine != "" {
				get["on"] = w.machine
			}
			if raw, err := s.Core.Call(ctx, "doc.get", get); err == nil {
				var res struct {
					Version uint64 `json:"version"`
				}
				_ = json.Unmarshal(raw, &res)
				w.version = res.Version
			}
		}
	}
}

// WriteDoc saves the model now, on the caller's goroutine.
func (s *Session) WriteDoc() { s.saveDoc() }

// SetActive selects a space and its first tab.
func (s *Session) SetActive(spaceID string) {
	s.mu.Lock()
	for _, sp := range s.spaces {
		if sp.ID == spaceID {
			s.active = spaceID
		}
	}
	s.mu.Unlock()
	s.changed()
}

// FocusPane focuses a pane of the shown tab.
func (s *Session) FocusPane(pane string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sp := range s.spaces {
		for _, t := range sp.Tabs {
			if t.Root.find(pane) == nil {
				continue
			}
			v := s.tabs[t.ID]
			if v == nil {
				v = &tabView{}
				s.tabs[t.ID] = v
			}
			if v.focused != pane {
				if v.focused != "" {
					v.before = append(v.before, v.focused)
				}
				v.focused = pane
			}
			s.active = sp.ID
			s.activeTab[sp.ID] = t.ID
			s.markSeenLocked(t.Root.find(pane).terminal())
			go func() { s.changed() }()
			return
		}
	}
}

// FocusTerminal focuses the pane showing a terminal, if it has one.
func (s *Session) FocusTerminal(addr string) bool {
	s.mu.Lock()
	for _, sp := range s.spaces {
		for _, t := range sp.Tabs {
			for _, b := range t.Root.blocks() {
				if b.terminal() == addr || s.query[b.ID] == addr {
					if v := s.tabs[t.ID]; v != nil {
						v.focused = b.ID
					}
					s.active = sp.ID
					s.activeTab[sp.ID] = t.ID
					s.markSeenLocked(addr)
					s.mu.Unlock()
					s.changed()
					return true
				}
			}
		}
	}
	s.mu.Unlock()
	return false
}

// markSeenLocked notes that the person looked at a terminal's finished agent.
func (s *Session) markSeenLocked(addr string) {
	if term := s.terminals[addr]; term != nil && term.Word() == "done" {
		s.seen[addr] = true
		if term.Labels == nil {
			term.Labels = map[string]string{}
		}
		term.Labels["herdr.seen"] = fmt.Sprint(term.Turn)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _ = s.Core.Call(ctx, "terminal.label", map[string]any{"terminal": addr, "labels": map[string]string{"herdr.seen": fmt.Sprint(term.Turn)}})
		}()
	}
}

// SelectTab shows a tab of its space.
func (s *Session) SelectTab(id string) {
	s.mu.Lock()
	for _, sp := range s.spaces {
		for _, t := range sp.Tabs {
			if t.ID == id {
				s.active = sp.ID
				s.activeTab[sp.ID] = id
			}
		}
	}
	s.mu.Unlock()
	s.changed()
}

// Tree returns a copy of a tab's pane tree, for drawing.
func (s *Session) Tree(tabID string) *Node {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sp := range s.spaces {
		for _, t := range sp.Tabs {
			if t.ID == tabID {
				return t.Root.clone()
			}
		}
	}
	return nil
}

// Tabs of the shown space.
func (s *Session) Tabs() []*Tab {
	s.mu.Lock()
	defer s.mu.Unlock()
	sp, _ := s.activeLocked()
	if sp == nil {
		return nil
	}
	return sp.Tabs
}

// SpaceOf returns the space a tab belongs to.
func (s *Session) SpaceOf(tabID string) *Space {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sp := range s.spaces {
		for _, t := range sp.Tabs {
			if t.ID == tabID {
				return sp
			}
		}
	}
	return nil
}

// osHome is the home folder.
func osHome() string {
	h, _ := os.UserHomeDir()
	return h
}

// ShortFolder shows a folder as the pane header does: ~ at home.
func ShortFolder(cwd string) string {
	if cwd == "" {
		return ""
	}
	h := osHome()
	if cwd == h {
		return "~"
	}
	if strings.HasPrefix(cwd, h+"/") {
		return "~" + strings.TrimPrefix(cwd, h)
	}
	return cwd
}
