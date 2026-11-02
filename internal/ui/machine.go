package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"

	"herdr-mygo/internal/herdr"
)

// sshHost is one Host from ~/.ssh/config that can be added as a machine.
type sshHost struct {
	Alias    string
	Aliases  []string
	Hostname string
	User     string
}

// parseSSHConfig lists the named hosts in an ssh config (patterns skipped).
func parseSSHConfig(text string) []sshHost {
	var hosts []sshHost
	cur := -1
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var key, value string
		if i := strings.IndexAny(line, " \t="); i > 0 {
			key, value = line[:i], strings.TrimSpace(line[i+1:])
			value = strings.TrimSpace(strings.TrimPrefix(value, "="))
		} else {
			continue
		}
		switch strings.ToLower(key) {
		case "host", "match":
			cur = -1
			if strings.ToLower(key) != "host" {
				continue
			}
			var names []string
			for _, name := range strings.Fields(value) {
				if name != "" && !strings.ContainsAny(name, "*?!") {
					names = append(names, name)
				}
			}
			if len(names) == 0 {
				continue
			}
			hosts = append(hosts, sshHost{Alias: names[0], Aliases: names[1:]})
			cur = len(hosts) - 1
		case "hostname":
			if cur >= 0 {
				hosts[cur].Hostname = value
			}
		case "user":
			if cur >= 0 {
				hosts[cur].User = value
			}
		}
	}
	return hosts
}

// machineAddState is the Add a machine screen: the hosts to offer, what each
// add is doing, and the terminal an add may need.
type machineAddState struct {
	loaded bool
	hosts  []sshHost
	query  string
	item   map[string]string // target -> "" | "adding" | "terminal" | an error
	term   string            // terminal showing an add, if any
	target string            // the machine being added in it
}

// addMachineState is a copy for drawing, safe to read outside the lock.
func (a *App) addMachineSnapshot() machineAddState {
	a.mu.Lock()
	defer a.mu.Unlock()
	st := a.mach
	st.hosts = append([]sshHost(nil), a.mach.hosts...)
	st.item = map[string]string{}
	for k, v := range a.mach.item {
		st.item[k] = v
	}
	return st
}

// setAddState records what adding one target is doing.
func (a *App) setAddState(target, state string) {
	a.mu.Lock()
	if a.mach.item == nil {
		a.mach.item = map[string]string{}
	}
	if state == "" {
		delete(a.mach.item, target)
	} else {
		a.mach.item[target] = state
	}
	a.mu.Unlock()
}

// loadSSHHosts reads ~/.ssh/config once, in the background.
func (a *App) loadSSHHosts() {
	a.mu.Lock()
	if a.mach.loaded {
		a.mu.Unlock()
		return
	}
	a.mach.loaded = true
	a.mu.Unlock()
	go func() {
		home, _ := os.UserHomeDir()
		text, _ := os.ReadFile(filepath.Join(home, ".ssh", "config"))
		hosts := parseSSHConfig(string(text))
		a.mu.Lock()
		a.mach.hosts = hosts
		a.mu.Unlock()
		a.Repaint()
	}()
}

// addMachine adds one target: one click when it works, a terminal when only
// a person can answer (a password, a host key, an install there).
func (a *App) addMachine(target string) {
	target = strings.TrimSpace(target)
	if target == "" {
		return
	}
	a.setAddState(target, "adding")
	a.Repaint()
	go func() {
		err := a.S.AddMachine(target, "")
		if err == nil {
			a.setAddState(target, "")
			a.mu.Lock()
			if a.mach.query == target {
				a.mach.query = ""
			}
			a.mu.Unlock()
			a.S.RefreshMachines()
			a.Repaint()
			return
		}
		if herdr.NeedsTerminal(err) {
			addr, termErr := a.S.AddMachineInTerminal(target)
			if termErr == nil {
				a.setAddState(target, "terminal")
				a.mu.Lock()
				a.mach.term = addr
				a.mach.target = target
				a.mu.Unlock()
				a.Repaint()
				return
			}
			a.setAddState(target, termErr.Error())
			a.Repaint()
			return
		}
		a.setAddState(target, err.Error())
		a.Repaint()
	}()
}

// machineAddOverlay is the Add a machine dialog.
func (a *App) machineAddOverlay(c *ui.Context) {
	t := a.T
	st := a.addMachineSnapshot()
	if st.term != "" {
		a.machineTerminal(c, st)
		return
	}
	a.loadSSHHosts()
	width := float32(560)
	rows := len(st.hosts)
	if rows > 6 {
		rows = 6
	}
	height := float32(250 + rows*40)
	if st.item[st.query] != "" {
		height += 34
	}
	panel := a.card(c, width, height).Absolute().Top((a.H - height) / 2).Left((a.W - width) / 2)
	panel.Padding(18, 18, 14, 18).Gap(0).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Add a machine").FontSize(15).FontWeight(600).TextColor(t.Foreground).Grow(1)
			ui.Text(c, "Esc").FontSize(11).TextColor(t.Dim)
		})
		ui.Text(c, "A VPS, a Mac mini, a box at home: Herdr reaches it over ssh, and agents there keep running while your laptop sleeps.").
			FontSize(12.8).TextColor(t.Muted).Wrap().Padding(6, 0, 0, 0)
		label := "From your ssh config"
		if st.loaded && len(st.hosts) == 0 {
			label = "Your ssh config lists no hosts"
		}
		ui.Text(c, label).FontSize(11.5).FontWeight(500).TextColor(t.Dim).Padding(16, 0, 6, 0)
		ui.Scroll(c).Height(float32(max(rows, 1) * 40)).Children(func() {
			ui.Column(c).FillWidth().Children(func() {
				if !st.loaded {
					ui.Row(c).Height(40).PaddingX(10).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "Reading ~/.ssh/config…").FontSize(12.5).TextColor(t.Dim)
					})
					return
				}
				for _, h := range st.hosts {
					a.machineRow(c, h.Alias, h.Alias, sshSub(h), st.item[h.Alias])
				}
			})
		})
		ui.Text(c, "Or type how ssh reaches it").FontSize(11.5).FontWeight(500).TextColor(t.Dim).Padding(14, 0, 6, 0)
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			field := ui.Row(c).Grow(1).Height(34).PaddingX(10).Radius(8).Border(1, t.Input).
				Background(t.Background).AlignItems(ui.Center)
			field.Children(func() {
				input := ui.TextInputBase(c, &a.mach.query).Grow(1).FontSize(13).TextColor(t.Foreground).
					Placeholder("user@host").AutoFocus()
				if input.Submitted() {
					a.addMachine(a.mach.query)
				}
			})
			a.smallButton(c, "Add", func() { a.addMachine(a.mach.query) })
		})
		if msg := st.item[st.query]; msg != "" {
			text := msg
			color := t.Blocked
			if msg == "adding" {
				text = "Connecting…"
				color = t.Muted
			} else if msg == "terminal" {
				text = "Waiting in its terminal"
				color = t.Blocked
			}
			ui.Text(c, text).FontSize(12).TextColor(color).Padding(8, 0, 0, 0).Wrap()
		}
	})
}

// sshSub is the "user@hostname" under a config alias.
func sshSub(h sshHost) string {
	if h.User != "" || h.Hostname != "" {
		return strings.TrimPrefix(h.User+"@"+h.Hostname, "@")
	}
	if len(h.Aliases) > 0 {
		return "also " + strings.Join(h.Aliases, ", ")
	}
	return ""
}

// machineRow is one host to add, with what its add is doing.
func (a *App) machineRow(c *ui.Context, target, label, sub, state string) {
	t := a.T
	added := false
	for _, m := range a.S.Machines() {
		if m.Target == target || m.Name == target {
			added = true
		}
	}
	row := ui.Row(c).Height(40).FillWidth().PaddingX(10).Gap(10).Radius(8).AlignItems(ui.Center)
	if row.Hovered() && !added && state == "" {
		row.Background(t.Hover)
	}
	row.Children(func() {
		a.icon(c, "server", 15, t.Muted)
		ui.Text(c, label).FontSize(13).TextColor(t.Foreground)
		ui.Text(c, sub).FontSize(12).TextColor(t.Dim).Grow(1).Ellipsis("…")
		switch {
		case added:
			ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
				a.icon(c, "circle-check", 13, t.Status("idle"))
				ui.Text(c, "Added").FontSize(12).TextColor(t.Status("idle"))
			})
		case state == "adding":
			ui.Text(c, "Connecting…").FontSize(12).TextColor(t.Muted)
		case state == "terminal":
			ui.Text(c, "Waiting in its terminal").FontSize(12).TextColor(t.Blocked)
		default:
			a.smallButton(c, "Add", func() { a.addMachine(target) })
		}
	})
	if added || state == "adding" || state == "terminal" {
		return
	}
	if row.Clicked() {
		a.addMachine(target)
	}
	if state != "" {
		ui.Text(c, state).FontSize(12).TextColor(t.Blocked).Padding(0, 35, 6, 35).Wrap()
	}
}

// smallButton is a compact button with a border, as the add rows use.
func (a *App) smallButton(c *ui.Context, label string, action func()) {
	t := a.T
	b := ui.Row(c).Height(26).PaddingX(10).Radius(7).Border(1, t.Input).Background(t.Raised).AlignItems(ui.Center)
	if b.Hovered() {
		b.Background(t.Hover)
	}
	b.Cursor(ui.CursorPointer)
	b.Children(func() {
		ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(t.Foreground)
	})
	if b.Clicked() {
		action()
	}
}

// machineTerminal shows the terminal an add needs answered in.
func (a *App) machineTerminal(c *ui.Context, st machineAddState) {
	t := a.T
	width := float32(640)
	height := float32(320)
	panel := a.card(c, width, height).Absolute().Top(90).Left((a.W - width) / 2).Clip()
	panel.Children(func() {
		head := ui.Row(c).Height(30).FillWidth().PaddingX(12).Gap(8).AlignItems(ui.Center).
			Background(t.Chrome)
		head.Children(func() {
			a.icon(c, "server", 14, t.Muted)
			ui.Text(c, fmt.Sprintf("Adding %s: answer it here", st.target)).FontSize(12.5).TextColor(t.Muted).Grow(1)
			close := ui.Box(c).Size(22, 22).Radius(6).Center()
			if close.Hovered() {
				close.Background(t.Hover)
			}
			close.Cursor(ui.CursorPointer)
			close.Children(func() { a.icon(c, "close", 12, t.Dim) })
			if close.Clicked() {
				a.mu.Lock()
				a.mach.term = ""
				a.mach.target = ""
				a.mu.Unlock()
				a.CloseOverlay()
			}
		})
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		ui.Box(c).Grow(1).FillWidth().Background(t.Background).Children(func() {
			a.terminalView(c, herdr.PaneInfo{Pane: "machine-add", Terminal: st.term})
		})
	})
}

// machinesSection is Settings' Machines page.
func (a *App) machinesSection(c *ui.Context) {
	t := a.T
	ui.Text(c, "Herdr reaches other computers over ssh; their terminals and agents keep running there.").
		FontSize(12.5).TextColor(t.Muted).Wrap()
	row := func(name, target, state string, connected bool, actions func()) {
		ui.Row(c).FillWidth().Height(40).Gap(10).AlignItems(ui.Center).Children(func() {
			a.icon(c, "server", 15, t.Muted)
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Text(c, name).FontSize(13).TextColor(t.Foreground)
				ui.Text(c, target).FontSize(11.5).TextColor(t.Dim).Ellipsis("…")
			})
			dot := t.Dim
			if connected {
				dot = t.Status("idle")
			}
			ui.Box(c).Size(6, 6).Radius(3).Background(dot)
			ui.Text(c, state).FontSize(11.5).TextColor(t.Dim)
			actions()
		})
	}
	row("This computer", "where this window runs", "connected", true, func() {})
	if add := ui.Row(c).Height(28).PaddingX(10).Radius(7).AlignItems(ui.Center); true {
		add.Cursor(ui.CursorPointer).Border(1, t.Input).Background(t.Raised)
		if add.Hovered() {
			add.Background(t.Hover)
		}
		add.Children(func() {
			a.icon(c, "plus", 12, t.Muted)
			ui.Text(c, "Add machine").FontSize(12).TextColor(t.Foreground).Padding(0, 0, 0, 6)
		})
		if add.Clicked() {
			a.OpenOverlay("machine-add")
		}
	}
	for _, m := range a.S.Machines() {
		m := m
		row(m.Name, m.Target, m.State, m.Connected(), func() {
			if m.Connected() {
				a.smallButton(c, "Remove", func() {
					go func() {
						if err := a.S.RemoveMachine(m.Name); err != nil {
							a.toast("Cannot remove the machine", err.Error(), "error")
						}
						a.S.RefreshMachines()
						a.Repaint()
					}()
				})
			} else {
				a.smallButton(c, "Connect", func() {
					go func() {
						if err := a.S.ConnectMachine(m.Name); err != nil {
							a.toast("Cannot connect", err.Error(), "error")
						}
						a.Repaint()
					}()
				})
			}
		})
		if m.Why != nil && *m.Why != "" {
			ui.Text(c, *m.Why).FontSize(11.5).TextColor(t.Blocked).Padding(0, 35, 0, 0).Wrap()
		}
	}
}
