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
	ui.Column(c).Fill().Background(t.Chrome).Children(func() {
		if !rail {
			header := ui.Row(c).Height(31).FillWidth().Padding(0, 10, 0, 16).AlignItems(ui.Center).Justify(ui.SpaceBetween)
			header.Children(func() {
				ui.Text(c, "Spaces").FontSize(11.5).TextColor(t.Dim)
				a.iconButton(c, "plus", 26, func() { a.newSpace() })
			})
		} else {
			ui.Row(c).Height(31).FillWidth().Children(func() {
				ui.Spacer(c)
				a.iconButton(c, "plus", 26, func() { a.newSpace() })
			})
		}
		ui.Scroll(c).Grow(1).Padding(0, 8, 14, 8).Children(func() {
			ui.Column(c).FillWidth().Gap(0).Children(func() {
				places := a.S.Places()
				machines := a.S.Machines()
				if len(places) == 0 && len(machines) == 0 {
					ui.Row(c).Height(28).PaddingX(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "Loading…").FontSize(12).TextColor(t.Dim)
					})
					return
				}
				if len(machines) == 0 {
					for i, sp := range places {
						a.spaceRow(c, sp, i, rail, places)
					}
					return
				}
				names := []string{""}
				for _, m := range machines {
					names = append(names, m.Name)
				}
				for _, name := range names {
					a.machineGroup(c, name, places, machines, rail)
				}
			})
		})
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

// spaceRow is one space: its avatar, name, and the unusual git after it.
func (a *App) spaceRow(c *ui.Context, sp *herdr.Space, index int, rail bool, places []*herdr.Space) {
	t := a.T
	if rail {
		row := ui.Row(c).Height(26).FillWidth().Gap(4).AlignItems(ui.Center).Radius(6).Justify(ui.Center)
		if row.Hovered() {
			row.Background(t.Hover)
		}
		row.Cursor(ui.CursorPointer)
		row.Children(func() {
			ui.Text(c, itoa(index+1)).FontSize(13).TextColor(t.Muted)
			ui.Text(c, "·").FontSize(13).TextColor(t.Dim)
		})
		if row.Clicked() {
			a.S.SelectSpace(0, sp.ID)
		}
		return
	}
	spaceName := a.S.SpaceName(sp)
	row := ui.Row(c).Height(30).FillWidth().PaddingX(8).Radius(8).AlignItems(ui.Center)
	if sp.Active {
		row.Background(t.Selected)
	} else if row.Hovered() {
		row.Background(t.Hover)
	}
	row.Cursor(ui.CursorPointer)
	row.Children(func() {
		ui.Row(c).FillWidth().Gap(9).AlignItems(ui.Center).Children(func() {
			name := spaceName
			mark := a.spaceMarkFor(sp, places)
			ui.Box(c).Size(18, 18).Radius(5).Background(mark).Center().Children(func() {
				letter := "?"
				if r := []rune(name); len(r) > 0 {
					letter = strings.ToUpper(string(r[0]))
				}
				ui.Text(c, letter).FontSize(10.5).FontWeight(700).TextColor(ui.RGB(255, 255, 255))
			})
			ui.Text(c, name).FontSize(13).TextColor(t.Foreground).MaxWidth(150).Ellipsis("…")
			if tail := a.gitTail(sp, name); tail != "" {
				ui.Text(c, tail).FontSize(11.5).TextColor(t.Dim).Grow(1).TextAlign(ui.End).Ellipsis("…")
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
				a.S.MoveSpace(d.ID, dropIndex(row, 30, from, sp.Index))
			}
		case d.Kind == "machine" && d.ID != machine && machine != "":
			// A machine let go over another machine's rows lands beside that
			// machine's group.
			machines := a.S.Machines()
			if from := machineAt(machines, d.ID); from > 0 {
				a.S.MoveMachine(d.ID, dropIndex(row, 30, from, machineAt(machines, machine)))
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
	ui.Column(c).Fill().Background(t.Chrome).BorderStartWidth(0).Children(func() {
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		if !rail {
			row := ui.Row(c).Height(41).FillWidth().Padding(14, 16, 5, 16).AlignItems(ui.Center).Justify(ui.SpaceBetween)
			row.Children(func() {
				ui.Text(c, "Agents").FontSize(11.5).FontWeight(500).TextColor(t.Dim)
				sort := ui.Box(c).Radius(4)
				if sort.Hovered() {
					sort.Background(t.Hover)
				}
				sort.Cursor(ui.CursorPointer)
				sort.Children(func() {
					a.icon(c, "clock", 14, t.Dim)
				})
			})
		}
		list := a.S.Agents()
		if len(list) == 0 && !rail {
			ui.Box(c).PaddingX(12).Children(func() {
				ui.Text(c, "No agent status yet. Agents show up here when they report their status. Older versions and agents inside tmux may not.").
					FontSize(12).TextColor(t.Dim).Wrap()
			})
		}
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).FillWidth().PaddingX(8).Children(func() {
				if rail {
					for i, ag := range list {
						a.agentRail(c, ag, i)
					}
					return
				}
				a.agentGroups(c, list)
			})
		})
		if rail {
			ui.Row(c).Height(28).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
				b := ui.Text(c, "»").FontSize(13).TextColor(t.Dim)
				if b.Clicked() {
					a.toggleSidebar()
				}
			})
		}
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
		a.agentStatus(c, ag.Status, false)
	})
	if row.Clicked() {
		a.S.FocusPane(ag.Pane)
	}
}

// agentRail is an agent in the narrow rail: its number and status.
func (a *App) agentRail(c *ui.Context, ag *herdr.Agent, index int) {
	t := a.T
	row := ui.Row(c).Height(26).FillWidth().PaddingX(8).Gap(4).AlignItems(ui.Center).Justify(ui.SpaceBetween)
	if row.Hovered() {
		row.Background(t.Hover)
	}
	row.Children(func() {
		ui.Text(c, itoa(index+1)).FontSize(13).TextColor(t.Muted)
		a.agentStatus(c, ag.Status, true)
	})
	if row.Clicked() {
		a.S.FocusPane(ag.Pane)
	}
}

// agentStatus is the working spinner, the Needs you pill, the idle ring or
// the done dot.
func (a *App) agentStatus(c *ui.Context, word string, compact bool) {
	t := a.T
	switch word {
	case "working":
		a.icon(c, "spinner", 12, t.Working)
	case "blocked":
		if compact {
			ui.Box(c).Size(14, 14).Radius(4).Center().Background(Alpha(t.Blocked, 0.15)).Children(func() {
				ui.Text(c, "!").FontSize(10.5).FontWeight(700).TextColor(t.Blocked)
			})
			return
		}
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
func (a *App) machineGroup(c *ui.Context, name string, places []*herdr.Space, machines []*herdr.Machine, rail bool) {
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
	if rail && name == "" && len(mine) > 0 {
		for i, sp := range mine {
			a.spaceRow(c, sp, i, rail, places)
		}
		return
	}
	head := ui.Row(c).Height(26).FillWidth().PaddingX(8).Gap(7).AlignItems(ui.Center)
	if head.Hovered() {
		head.Background(t.Hover)
	}
	head.Cursor(ui.CursorPointer)
	head.Children(func() {
		icon := "monitor"
		if name != "" {
			icon = "server"
		}
		a.icon(c, icon, 13, t.Dim)
		if !rail {
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
				plus := ui.Box(c).Size(16, 16).Radius(4).Center()
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
		}
	})
	if head.Clicked() {
		if folded {
			delete(a.foldedMachines, name)
		} else {
			a.foldedMachines[name] = true
		}
	}
	// A remote machine's header drags its group up or down the sidebar, and
	// takes a space dragged onto it at the top of its list.
	if name != "" && !rail {
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
		row := ui.Row(c).Height(28).PaddingX(8).Gap(7).AlignItems(ui.Center).Radius(7)
		if row.Hovered() {
			row.Background(t.Hover)
		}
		row.Cursor(ui.CursorPointer)
		row.Children(func() {
			a.icon(c, "plus", 12, t.Dim)
			if !rail {
				ui.Text(c, "New space there").FontSize(12).TextColor(t.Dim)
			}
		})
		if row.Clicked() {
			a.newSpaceOn(name)
		}
		return
	}
	for i, sp := range mine {
		a.spaceRow(c, sp, i, rail, places)
	}
}
