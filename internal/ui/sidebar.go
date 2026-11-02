package ui

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"herdr-mygo/internal/herdr"
)

// spaceColors are the avatar colors Herdr gives spaces, by ansi name.
var spaceColors = []string{"ansi6", "ansi3", "ansi4", "ansi5", "ansi2", "ansi1"}

// spaces draws the Spaces list of the sidebar.
func (a *App) spaces(c *ui.Context, rail bool) {
	t := a.T
	statuses := a.spaceStatuses()
	ui.Column(c).Fill().Background(t.Chrome).Children(func() {
		if !rail {
			header := ui.Row(c).Height(31).FillWidth().Padding(0, 10, 0, 16).AlignItems(ui.Center).Justify(ui.SpaceBetween)
			header.Children(func() {
				ui.Text(c, "Spaces").FontSize(11.5).TextColor(t.Dim)
				a.iconButton(c, "plus", 26, func() { a.newSpace() })
			})
		} else {
			ui.Row(c).Height(31).FillWidth().Justify(ui.Center).Children(func() {
				a.iconButton(c, "plus", 26, func() { a.newSpace() })
			})
			a.railAlert(c)
		}
		ui.Scroll(c).Grow(1).Padding(0, 8, 14, 8).Children(func() {
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				places := a.S.Places()
				machines := a.S.Machines()
				if len(places) == 0 && len(machines) == 0 {
					ui.Row(c).Height(28).PaddingX(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "Loading…").FontSize(12).TextColor(t.Dim)
					})
					return
				}
				if rail {
					prev := ""
					for i, sp := range places {
						machine := herdr.SpaceMachine(sp)
						if i > 0 && machine != prev {
							a.railDivider(c)
						}
						a.spaceRow(c, sp, true, places, statuses)
						prev = machine
					}
					return
				}
				if len(machines) == 0 {
					for _, sp := range places {
						a.spaceRow(c, sp, false, places, statuses)
					}
					return
				}
				names := []string{""}
				for _, m := range machines {
					names = append(names, m.Name)
				}
				for _, name := range names {
					a.machineGroup(c, name, places, machines, statuses)
				}
			})
		})
	})
}

// spaceStatuses names the most urgent agent of each space, by space id.
func (a *App) spaceStatuses() map[string]string {
	out := map[string]string{}
	for _, ag := range a.S.Agents() {
		if _, ok := out[ag.Space]; !ok {
			out[ag.Space] = ag.Status
		}
	}
	return out
}

// railAlert is the blocked count at the top of the rail; clicking it goes to
// the first agent that needs you.
func (a *App) railAlert(c *ui.Context) {
	t := a.T
	n := 0
	var first *herdr.Agent
	for _, ag := range a.S.Agents() {
		if ag.Status == "blocked" {
			n++
			if first == nil {
				first = ag
			}
		}
	}
	if n == 0 {
		return
	}
	ui.Row(c).Height(28).FillWidth().Justify(ui.Center).Children(func() {
		pill := ui.Row(c).Height(22).PaddingX(6).Gap(3).Radius(6).Background(Alpha(t.Blocked, 0.15)).AlignItems(ui.Center)
		pill.Cursor(ui.CursorPointer)
		if pill.Hovered() {
			pill.Background(Alpha(t.Blocked, 0.24))
		}
		pill.Children(func() {
			ui.Text(c, "!").FontSize(10.5).FontWeight(700).TextColor(t.Blocked)
			ui.Text(c, itoa(n)).FontSize(10.5).FontWeight(600).TextColor(t.Blocked)
		})
		if pill.Clicked() && first != nil {
			a.S.FocusPane(first.Pane)
		}
	})
}

// sideDrag is a sidebar row being dragged: a space (Kind "space", From its
// machine's name) or a machine's header (Kind "machine", ID its name).
type sideDrag struct {
	Kind string
	ID   string
	From string
}

// dropIndex is the place (1-based) MoveSpace and MoveMachine want: the half
// of the row the pointer let go on picks before or after it, and a target
// lower down moves up a place as the dragged one leaves the list.
func dropIndex(row ui.Element, height float32, from, to int) int {
	_, y, _ := row.PointerPosition()
	return dropIndexAt(y, height, from, to)
}

// dropIndexAt is dropIndex once the pointer's height in the row is known.
func dropIndexAt(y, height float32, from, to int) int {
	at := to - 1
	if y > height/2 {
		at++
	}
	if from < to {
		at--
	}
	return at + 1
}

// shownIndex is the 1-based place a space has in the sidebar, 0 when it is
// not there.
func shownIndex(places []*herdr.Space, id string) int {
	for _, sp := range places {
		if sp.ID == id {
			return sp.Index
		}
	}
	return 0
}

// machineAt is a machine's place among the sidebar's groups (1 = first
// remote machine), 0 when it is not there.
func machineAt(machines []*herdr.Machine, name string) int {
	for i, m := range machines {
		if m.Name == name {
			return i + 1
		}
	}
	return 0
}

// spaceRow is one space: its avatar, name, status dot and the unusual git
// after it. In the rail it is just the avatar and its status badge.
func (a *App) spaceRow(c *ui.Context, sp *herdr.Space, rail bool, places []*herdr.Space, statuses map[string]string) {
	if rail {
		a.railSpaceRow(c, sp, places, statuses[sp.ID])
		return
	}
	t := a.T
	spaceName := a.S.SpaceName(sp)
	row := ui.Row(c).Height(26).FillWidth().PaddingX(8).Radius(8).AlignItems(ui.Center)
	if sp.Active {
		row.Background(t.Selected)
	} else if row.Hovered() {
		row.Background(t.Hover)
	}
	row.Cursor(ui.CursorPointer)
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			name := spaceName
			mark := a.spaceMarkFor(sp, places)
			ui.Box(c).Size(14, 14).Radius(4).Background(mark).Center().Children(func() {
				letter := "?"
				if r := []rune(name); len(r) > 0 {
					letter = strings.ToUpper(string(r[0]))
				}
				ui.Text(c, letter).FontSize(8.5).FontWeight(700).TextColor(ui.RGB(255, 255, 255))
			})
			ui.Text(c, name).FontSize(12.5).TextColor(t.Foreground).MaxWidth(110).Ellipsis("…")
			if status := statuses[sp.ID]; status != "" {
				ui.Box(c).Size(6, 6).Radius(3).Background(t.Status(status))
			}
			ui.Spacer(c)
			if tail := a.gitTail(sp, name); tail != "" {
				a.gitChip(c, tail)
			}
		})
	})
	if row.Clicked() {
		a.S.SelectSpace(0, sp.ID)
	}
	// Dragging a row reorders the spaces of its own machine; a space never
	// leaves the machine it runs on.
	machine := herdr.SpaceMachine(sp)
	row.Drag(sideDrag{Kind: "space", ID: sp.ID, From: machine})
	// The node takes one kind of value, so space rows and machine headers
	// share the one sideDrag type and check what was dropped.
	if d, ok := ui.Drop[sideDrag](row); ok {
		switch {
		case d.Kind == "space" && d.ID != sp.ID && d.From == machine:
			if from := shownIndex(places, d.ID); from > 0 {
				a.S.MoveSpace(d.ID, dropIndex(row, 26, from, sp.Index))
			}
		case d.Kind == "machine" && d.ID != machine && machine != "":
			// A machine let go over another machine's rows lands beside that
			// machine's group.
			machines := a.S.Machines()
			if from := machineAt(machines, d.ID); from > 0 {
				a.S.MoveMachine(d.ID, dropIndex(row, 26, from, machineAt(machines, machine)))
			}
		}
	}
	if d, ok := ui.DragOver[sideDrag](row); ok && d.Kind == "space" && d.ID != sp.ID && d.From == machine {
		row.Border(1, t.Accent)
	}
	if row.RightClicked() {
		x, y, _ := row.PointerPosition()
		b := row.Bounds()
		id := sp.ID
		a.openMenuAt(b.X+x, b.Y+y, []MenuItem{
			{Label: "Rename", Icon: "square-pen", Do: func() {
				a.Prompt("Rename space", "Leave empty for where its first terminal is", sp.Name, "space.rename", id)
			}},
			{Label: "New tab", Icon: "plus", Do: func() { a.S.SelectSpace(0, id); a.newTab() }},
			{Label: "Close", Icon: "close", Do: func() { a.S.CloseSpace(id) }},
		})
	}
}

// railDivider separates one machine's spaces from the next machine's in the
// rail, where there is no room for the machine header.
func (a *App) railDivider(c *ui.Context) {
	ui.Row(c).Height(9).FillWidth().Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Width(32).Height(1).Background(a.T.Border)
	})
}

// railSpaceRow is a space in the rail: its colored initial, with a badge for
// the most urgent agent inside it.
func (a *App) railSpaceRow(c *ui.Context, sp *herdr.Space, places []*herdr.Space, status string) {
	t := a.T
	ui.Row(c).Height(34).FillWidth().Justify(ui.Center).AlignItems(ui.Center).Children(func() {
		box := ui.Box(c).Size(30, 30).Radius(8)
		if sp.Active {
			box.Background(t.Selected)
		} else if box.Hovered() {
			box.Background(t.Hover)
		}
		box.Cursor(ui.CursorPointer)
		box.Children(func() {
			name := a.S.SpaceName(sp)
			mark := a.spaceMarkFor(sp, places)
			ui.Box(c).Size(22, 22).Radius(6).Background(mark).Center().Children(func() {
				letter := "?"
				if r := []rune(name); len(r) > 0 {
					letter = strings.ToUpper(string(r[0]))
				}
				ui.Text(c, letter).FontSize(12).FontWeight(700).TextColor(ui.RGB(255, 255, 255))
			})
			if status != "" {
				ui.Box(c).Absolute().Right(1).Bottom(1).Size(8, 8).Radius(4).
					Background(t.Status(status)).Border(1.5, t.Chrome)
			}
		})
		if box.Clicked() {
			a.S.SelectSpace(0, sp.ID)
		}
	})
}

// gitChip is a space's unusual git as one compact chip.
func (a *App) gitChip(c *ui.Context, tail string) {
	t := a.T
	ui.Row(c).Height(17).PaddingX(4).Gap(3).Radius(4).Border(1, t.Border).AlignItems(ui.Center).Children(func() {
		a.icon(c, "git-branch", 9, t.Dim)
		ui.Text(c, tail).FontSize(10).Font("JetBrains Mono").TextColor(t.Muted).MaxWidth(92).Ellipsis("…")
	})
}

// gitTail is what the row says after the name: repo, branch, ahead, behind.
func (a *App) gitTail(sp *herdr.Space, name string) string {
	info := a.S.GitOf(sp)
	if info.Branch == "" && info.Repo == "" {
		return ""
	}
	branch := info.Branch
	if branch == "HEAD" {
		branch = "detached"
	}
	if strings.EqualFold(branch, name) || branch == "master" || branch == "main" {
		branch = ""
	}
	repo := info.Repo
	if strings.EqualFold(repo, name) {
		repo = ""
	}
	var parts []string
	if repo != "" {
		parts = append(parts, repo)
	}
	if branch != "" {
		parts = append(parts, branch)
	}
	out := strings.Join(parts, " · ")
	if info.Ahead > 0 {
		out += " ↑" + itoa(info.Ahead)
	}
	if info.Behind > 0 {
		out += " ↓" + itoa(info.Behind)
	}
	return out
}

// agents draws the agents list of the sidebar.
func (a *App) agents(c *ui.Context, rail bool) {
	t := a.T
	list := a.S.Agents()
	ui.Column(c).Fill().Background(t.Chrome).Children(func() {
		if !rail {
			blocked, working := 0, 0
			for _, ag := range list {
				switch ag.Status {
				case "blocked":
					blocked++
				case "working":
					working++
				}
			}
			row := ui.Row(c).Height(41).FillWidth().Padding(14, 16, 5, 16).Gap(8).AlignItems(ui.Center)
			row.Children(func() {
				ui.Text(c, "Agents").FontSize(11.5).FontWeight(500).TextColor(t.Dim)
				ui.Spacer(c)
				a.agentCount(c, "blocked", blocked)
				a.agentCount(c, "working", working)
				chevron := ui.Box(c).Size(20, 20).Radius(4).Center()
				if chevron.Hovered() {
					chevron.Background(t.Hover)
				}
				chevron.Cursor(ui.CursorPointer)
				chevron.Children(func() {
					icon := "chevron-down"
					if a.agentsCollapsed {
						icon = "chevron-right"
					}
					a.icon(c, icon, 12, t.Dim)
				})
				if chevron.Clicked() {
					a.agentsCollapsed = !a.agentsCollapsed
					a.Repaint()
				}
			})
		}
		if a.agentsCollapsed && !rail {
			return
		}
		if len(list) == 0 && !rail {
			ui.Box(c).PaddingX(12).Children(func() {
				ui.Text(c, "No agent status yet. Older agents or tmux may not report.").
					FontSize(12).TextColor(t.Dim).Wrap()
			})
		}
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).FillWidth().PaddingX(8).Children(func() {
				if rail {
					for _, ag := range list {
						a.agentRail(c, ag)
					}
					return
				}
				a.agentGroups(c, list)
			})
		})
	})
}

// agentCount is a status dot and count for the agents header.
func (a *App) agentCount(c *ui.Context, word string, n int) {
	if n == 0 {
		return
	}
	t := a.T
	ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(t.Status(word))
		ui.Text(c, itoa(n)).FontSize(11).TextColor(t.Dim)
	})
}

var agentSections = []struct {
	word  string
	label string
}{
	{"blocked", "Needs you"},
	{"done", "Done"},
	{"working", "Working"},
	{"idle", "Idle"},
}

// agentGroups draws the agents by priority, under section headers.
func (a *App) agentGroups(c *ui.Context, list []*herdr.Agent) {
	t := a.T
	for _, section := range agentSections {
		var mine []*herdr.Agent
		for _, ag := range list {
			if ag.Status == section.word {
				mine = append(mine, ag)
			}
		}
		if len(mine) == 0 {
			continue
		}
		head := ui.Row(c).Height(30).FillWidth().Padding(8, 8, 0, 8).Gap(6).AlignItems(ui.Center)
		head.Children(func() {
			ui.Text(c, section.label).FontSize(11).FontWeight(500).TextColor(t.Status(section.word))
			ui.Text(c, itoa(len(mine))).FontSize(11).TextColor(t.Dim)
			ui.Spacer(c)
		})
		for _, ag := range mine {
			a.agentRow(c, ag)
		}
	}
}

// agentRow is one agent.
func (a *App) agentRow(c *ui.Context, ag *herdr.Agent) {
	t := a.T
	row := ui.Row(c).Height(30).FillWidth().PaddingX(8).Gap(9).Radius(8).AlignItems(ui.Center)
	if ag.Focused {
		row.Background(t.Selected)
	} else if row.Hovered() {
		row.Background(t.Hover)
	}
	row.Cursor(ui.CursorPointer)
	row.Children(func() {
		a.programIcon(c, ag.Program, 16, t.Foreground)
		name := ag.SpaceName
		if !strings.HasPrefix(ag.TabName, ag.SpaceName) && ag.TabName != "" && ag.TabName != "1" {
			name += " · " + ag.TabName
		}
		color := t.Foreground
		if ag.Status == "idle" && !ag.Focused {
			color = t.Muted
		}
		ui.Text(c, name).FontSize(13).TextColor(color).Grow(1).Ellipsis("…")
		a.agentStatus(c, ag.Status)
	})
	if row.Clicked() {
		a.S.FocusPane(ag.Pane)
	}
}

// agentRail is an agent in the narrow rail: its program, with a status badge.
func (a *App) agentRail(c *ui.Context, ag *herdr.Agent) {
	t := a.T
	row := ui.Row(c).Height(30).FillWidth().Justify(ui.Center).AlignItems(ui.Center)
	if row.Hovered() {
		row.Background(t.Hover)
	}
	row.Cursor(ui.CursorPointer)
	row.Children(func() {
		ui.Box(c).Size(24, 24).Children(func() {
			ui.Box(c).Fill().Center().Children(func() {
				a.programIcon(c, ag.Program, 16, t.Foreground)
			})
			if ag.Status != "" {
				ui.Box(c).Absolute().Right(-1).Bottom(-1).Size(7, 7).Radius(4).
					Background(t.Status(ag.Status)).Border(1.5, t.Chrome)
			}
		})
	})
	if row.Clicked() {
		a.S.FocusPane(ag.Pane)
	}
}

// agentStatus is the working spinner, the Needs you pill, the idle ring or
// the done dot.
func (a *App) agentStatus(c *ui.Context, word string) {
	t := a.T
	switch word {
	case "working":
		a.spinner(c, 12, t.Working)
	case "blocked":
		ui.Box(c).Height(18).PaddingX(6).Radius(5).Center().Background(Alpha(t.Blocked, 0.15)).Children(func() {
			ui.Text(c, "Needs you").FontSize(10.5).FontWeight(600).TextColor(t.Blocked)
		})
	case "idle":
		ui.Box(c).Size(7, 7).Radius(4).Border(1.5, t.Idle)
	case "done":
		ui.Box(c).Size(7, 7).Radius(4).Background(t.Done)
	}
}

// newSpace starts a space in the focused folder, on the shown machine.
func (a *App) newSpace() {
	sp, _ := a.S.Active()
	on := ""
	if sp != nil && sp.Machine != nil {
		on = *sp.Machine
	}
	a.newSpaceOn(on)
}

// newSpaceOn starts a space on one machine ("" is this computer).
func (a *App) newSpaceOn(machine string) {
	go func() {
		if _, _, err := a.S.NewSpace("", "", machine, nil); err != nil {
			a.toast("Cannot start a space", err.Error(), "error")
		}
	}()
}

// itoa is fmt.Sprint for small positive numbers.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// spaceMarkFor is a space's avatar color: the one fewest spaces use, as
// Herdr's mods agree on (a space called "~" keeps the first).
func (a *App) spaceMarkFor(sp *herdr.Space, places []*herdr.Space) ui.Color {
	assigned := map[string]string{}
	for _, other := range places {
		// A space called ~ is not given a color, as in Herdr's spaces mod.
		if other.Name == "~" {
			if other.ID == sp.ID {
				return ui.Hex(markColor(spaceColors[0]))
			}
			continue
		}
		uses := func(c string) int {
			n := 0
			for _, v := range assigned {
				if v == c {
					n++
				}
			}
			return n
		}
		best := spaceColors[0]
		for _, c := range spaceColors {
			if uses(c) < uses(best) {
				best = c
			}
		}
		if other.ID == sp.ID {
			return ui.Hex(markColor(best))
		}
		assigned[other.ID] = best
	}
	return ui.Hex(markColor(spaceColors[0]))
}

// markColor is the strong color a space avatar shows for an ansi name.
func markColor(name string) string {
	switch name {
	case "ansi6":
		return "#22a99a"
	case "ansi3":
		return "#e3814f"
	case "ansi4":
		return "#5b8def"
	case "ansi5":
		return "#8b7cf6"
	case "ansi2":
		return "#3fae6a"
	case "ansi1":
		return "#d9649a"
	}
	return "#22a99a"
}

// machineGroup draws one machine's spaces under its header, once a second
// machine is known.
func (a *App) machineGroup(c *ui.Context, name string, places []*herdr.Space, machines []*herdr.Machine, statuses map[string]string) {
	t := a.T
	var mine []*herdr.Space
	for _, sp := range places {
		if herdr.SpaceMachine(sp) == name {
			mine = append(mine, sp)
		}
	}
	rank := 0
	var info *herdr.Machine
	for i, m := range machines {
		if m.Name == name {
			info = m
			rank = i + 1
		}
	}
	connected := name == "" || (info != nil && info.Connected())
	folded := a.foldedMachines[name]
	head := ui.Row(c).Height(26).FillWidth().PaddingX(8).Gap(7).Radius(8).AlignItems(ui.Center)
	head.Cursor(ui.CursorPointer)
	var plus ui.Element
	head.Children(func() {
		icon := "monitor"
		if name != "" {
			icon = "server"
		}
		a.icon(c, icon, 13, t.Dim)
		label := name
		if label == "" {
			label = "This computer"
		}
		ui.Text(c, label).FontSize(11).TextColor(t.Dim).Grow(1).Ellipsis("…")
		if !connected {
			state := "asleep"
			if info != nil && info.State != "" {
				state = strings.ReplaceAll(info.State, "_", " ")
			}
			ui.Text(c, state).FontSize(11).TextColor(t.Dim)
		} else {
			plus = ui.Box(c).Size(16, 16).Radius(4).Center()
			if plus.Hovered() {
				plus.Background(t.Hover)
			}
			plus.Cursor(ui.CursorPointer)
			plus.Children(func() { a.icon(c, "plus", 11, t.Muted) })
			if plus.Clicked() {
				a.newSpaceOn(name)
			}
		}
		dot := t.Dim
		if connected {
			dot = t.Status("idle")
		}
		ui.Box(c).Size(6, 6).Radius(3).Background(dot)
		chevron := "chevron-down"
		if folded {
			chevron = "chevron-right"
		}
		a.icon(c, chevron, 11, t.Dim)
	})
	// The plus is its own hover surface; light the header only outside it, so
	// the two never stack into a brighter patch.
	if head.Hovered() && !(connected && plus.Hovered()) {
		head.Background(t.Hover)
	}
	if head.Clicked() {
		if folded {
			delete(a.foldedMachines, name)
		} else {
			a.foldedMachines[name] = true
		}
	}
	// A remote machine's header drags its group up or down the sidebar, and
	// takes a space dragged onto it at the top of its list.
	if name != "" {
		head.Drag(sideDrag{Kind: "machine", ID: name})
		if d, ok := ui.Drop[sideDrag](head); ok {
			switch {
			case d.Kind == "machine" && d.ID != name:
				if from := machineAt(machines, d.ID); from > 0 {
					a.S.MoveMachine(d.ID, dropIndex(head, 26, from, rank))
				}
			case d.Kind == "space" && d.From == name && len(mine) > 0:
				a.S.MoveSpace(d.ID, mine[0].Index)
			}
		}
		if d, ok := ui.DragOver[sideDrag](head); ok {
			if (d.Kind == "machine" && d.ID != name) || (d.Kind == "space" && d.From == name) {
				head.Border(1, t.Accent)
			}
		}
	}
	if folded {
		return
	}
	if len(mine) == 0 {
		if !a.S.MachineReady(name) {
			if !connected {
				return
			}
			ui.Row(c).Height(28).PaddingX(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Loading…").FontSize(12).TextColor(t.Dim)
			})
			return
		}
		row := ui.Row(c).Height(26).PaddingX(8).Gap(7).AlignItems(ui.Center).Radius(8)
		if row.Hovered() {
			row.Background(t.Hover)
		}
		row.Cursor(ui.CursorPointer)
		row.Children(func() {
			a.icon(c, "plus", 12, t.Dim)
			ui.Text(c, "New space there").FontSize(12).TextColor(t.Dim)
		})
		if row.Clicked() {
			a.newSpaceOn(name)
		}
		return
	}
	for _, sp := range mine {
		a.spaceRow(c, sp, false, places, statuses)
	}
}
