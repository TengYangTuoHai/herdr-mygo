// Package ui draws the window: the Herdr frame rebuilt in Go on MyGo's
// native UI, with herdr-core running the terminals behind it.
package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/terminal"
	"github.com/egoist/mygo/ui"

	"herdr-mygo/internal/herdr"
)

// App is the window's state: the core's world, what the window shows of it,
// and the terminal views attached to the core's terminals.
type App struct {
	S   *herdr.Session
	T   *Theme
	Win *mygo.Window

	W, H float32

	// Zoom is how large the app draws, as settings.json's zoom.
	Zoom float32
	// SidebarWidth is the sidebar's width in DIPs, 0 when it is the rail.
	SidebarWidth float32
	// splitRatio is the spaces list's share of the sidebar's two lists;
	// agentsCollapsed folds the agents list to its header.
	splitRatio      float32
	agentsCollapsed bool

	mu       sync.Mutex
	terms    map[string]*paneTerm
	starting map[string]bool

	focusPane string
	overlay   string
	err       error

	palette PaletteState
	notes   []Toast

	prompt    PromptState
	settings  settingsState
	picker    pickerState
	menuItems []MenuItem
	menuTitle string
	menuX     float32
	menuY     float32

	panel         string
	files         filesPanel
	changes       changesPanel
	commitMessage string

	mach           machineAddState
	foldedMachines map[string]bool

	fsMu   sync.Mutex
	remote map[string]*remoteDir
}

type paneTerm struct {
	key  string
	pane string
	addr string
	term *terminal.Terminal
	conn *herdr.Attach
}

// NewApp makes the window's state.
func NewApp(s *herdr.Session, theme *Theme) *App {
	return &App{
		S: s, T: theme,
		Zoom:           1,
		SidebarWidth:   268,
		splitRatio:     0.5,
		terms:          map[string]*paneTerm{},
		starting:       map[string]bool{},
		foldedMachines: map[string]bool{},
	}
}

// Repaint asks the window for a frame.
func (a *App) Repaint() {
	if a.Win != nil {
		a.Win.Invalidate()
	}
}

// View builds the whole window.
func (a *App) View(c *ui.Context) {
	t := a.T
	if a.S == nil {
		ui.Box(c).Fill().Background(t.Canvas).Center().Children(func() {
			ui.Column(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Herdr").FontSize(18).FontWeight(600).TextColor(t.Foreground)
				msg := "Connecting to herdr-core…"
				if a.err != nil {
					msg = a.err.Error()
				}
				ui.Text(c, msg).FontSize(12.5).TextColor(t.Dim)
			})
		})
		return
	}
	w, h := c.Size()
	a.W, a.H = w, h
	if a.Zoom != 1 {
		c.SetZoom(a.Zoom)
		w, h = c.Size()
		a.W, a.H = w, h
	}
	a.keys(c)
	if a.overlay != "" && c.Shortcut(0, ui.KeyEscape) {
		a.CloseOverlay()
	}
	a.collect()
	ui.Row(c).Fill().Background(t.Canvas).Font(UIFont).AlignItems(ui.Start).Children(func() {
		a.sidebar(c)
		ui.Box(c).Width(1).FillHeight().Background(t.Border)
		a.work(c)
	})
	a.resizeHandle(c)
	a.overlays(c)
}

// sidebar draws the brand row, the spaces and agents lists, and the footer.
func (a *App) sidebar(c *ui.Context) {
	t := a.T
	width := a.SidebarWidth
	rail := width < 160
	footHeight := a.footerHeight(rail)
	// listHeight is what the spaces and agents lists share, the divider aside.
	listHeight := a.H - 46 - footHeight - 1
	if listHeight < 90 {
		listHeight = 90
	}
	spacesHeight := listHeight / 2
	if !rail {
		if a.agentsCollapsed {
			spacesHeight = listHeight - 41
		} else {
			minSpaces, maxSpaces := splitLimits(listHeight)
			spacesHeight = listHeight * a.splitRatio
			if spacesHeight < minSpaces {
				spacesHeight = minSpaces
			}
			if spacesHeight > maxSpaces {
				spacesHeight = maxSpaces
			}
		}
	}
	ui.Column(c).Width(width).FillHeight().Background(t.Chrome).Children(func() {
		a.brand(c, rail)
		ui.Box(c).Height(spacesHeight).FillWidth().Children(func() { a.spaces(c, rail) })
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		ui.Box(c).Grow(1).FillWidth().Children(func() { a.agents(c, rail) })
		a.footer(c, rail)
	})
	if !rail && !a.agentsCollapsed {
		a.splitHandle(c, spacesHeight, listHeight)
	}
}

// splitHandle drags the line between the spaces and agents lists.
func (a *App) splitHandle(c *ui.Context, spacesHeight, listHeight float32) {
	t := a.T
	ui.Overlay(c, func() {
		handle := ui.Box(c).Absolute().Top(46 + spacesHeight - 3).Left(0).
			Width(a.SidebarWidth).Height(6).Cursor(ui.CursorResizeNS)
		if handle.Hovered() {
			handle.Background(Alpha(t.Accent, 0.35))
		}
		if dy, _, ok := handle.Dragged(); ok && listHeight > 0 {
			minSpaces, maxSpaces := splitLimits(listHeight)
			next := spacesHeight + dy
			if next < minSpaces {
				next = minSpaces
			}
			if next > maxSpaces {
				next = maxSpaces
			}
			a.splitRatio = next / listHeight
		}
	})
}

// splitLimits keeps the spaces list tall enough to read, and leaves the
// agents header and one section of its list showing, until the window is
// too short for both.
func splitLimits(listHeight float32) (min, max float32) {
	min = 90
	max = listHeight - 101
	if max < min {
		max = listHeight - 41
		if max < 0 {
			max = 0
		}
		min = max
	}
	return min, max
}

// footerHeight is the sidebar footer's height; the rail adds the expand row.
func (a *App) footerHeight(rail bool) float32 {
	if rail {
		return 97
	}
	return 69
}

// brand is the row with the logo, search and the sidebar toggle.
func (a *App) brand(c *ui.Context, rail bool) {
	t := a.T
	left := float32(14)
	if !rail && isMac() {
		left = 76
	}
	row := ui.Row(c).Height(46).FillWidth().Background(t.Chrome).Padding(0, 10, 0, left).Gap(4).AlignItems(ui.Center).DragWindow()
	row.Children(func() {
		if !rail {
			ui.Row(c).Grow(1).Gap(8).AlignItems(ui.Center).Children(func() {
				a.logo(c, 20)
				ui.Text(c, "Herdr").FontSize(13.5).FontWeight(600).TextColor(t.Foreground)
			})
			a.iconButton(c, "search", 26, func() { a.OpenOverlay("picker") })
			a.iconButton(c, "sidebar", 26, a.toggleSidebar)
			return
		}
		ui.Spacer(c)
		if !isMac() {
			a.iconButton(c, "sidebar", 26, a.toggleSidebar)
		}
	})
}

func (a *App) logo(c *ui.Context, size float32) {
	data, err := iconPNG("img/herdr.png")
	if err != nil {
		return
	}
	bm, err := ui.DecodeBitmap(data)
	if err != nil {
		return
	}
	ui.Box(c).Size(size, size).Radius(5).Clip().Children(func() {
		ui.Image(c, bm).Size(size, size).Fit(ui.Contain)
	})
}

// iconButton is a small square button of the chrome.
func (a *App) iconButton(c *ui.Context, name string, size float32, onClick func()) {
	t := a.T
	b := ui.Box(c).Size(size, size).Radius(6).Center()
	if b.Hovered() {
		b.Background(t.Hover)
	}
	if b.Clicked() && onClick != nil {
		onClick()
	}
	b.Cursor(ui.CursorPointer)
	b.Children(func() {
		color := t.Muted
		if b.Hovered() {
			color = t.Foreground
		}
		a.icon(c, name, size-11, color)
	})
}

// icon draws one of the app's SVGs.
func (a *App) icon(c *ui.Context, name string, size float32, color ui.Color) {
	svg := iconSVG(name)
	if svg == nil {
		return
	}
	ui.Icon(c, svg).Size(size, size).TextColor(color)
}

// programIcon draws what a terminal runs, as the app's program table says:
// a full-color logo in its own colors, or a one-color glyph in color.
func (a *App) programIcon(c *ui.Context, p herdr.ProgramInfo, size float32, color ui.Color) {
	svg := iconSVG("program:" + p.Icon)
	if svg == nil {
		a.icon(c, "terminal", size, color)
		return
	}
	if !p.Tint {
		ui.Image(c, svg).Size(size, size).Fit(ui.Contain)
		return
	}
	ui.Icon(c, svg).Size(size, size).TextColor(color)
}

func (a *App) toggleSidebar() {
	if a.SidebarWidth < 160 {
		a.SidebarWidth = 268
	} else {
		a.SidebarWidth = 76
	}
	a.Repaint()
}

// footer is Settings, Add machine, and the rail's expand row.
func (a *App) footer(c *ui.Context, rail bool) {
	t := a.T
	ui.Column(c).Height(a.footerHeight(rail)).FillWidth().Background(t.Chrome).Children(func() {
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		ui.Column(c).Grow(1).FillWidth().Padding(4, 8, 8, 8).Children(func() {
			a.footerRow(c, "settings", "Settings", false, rail, func() { a.OpenOverlay("settings") })
			a.footerRow(c, "plus", "Add machine", true, rail, func() { a.OpenOverlay("machine-add") })
			if rail {
				a.footerRow(c, "chevron-right", "Expand sidebar", true, rail, a.toggleSidebar)
			}
		})
	})
}

func (a *App) footerRow(c *ui.Context, icon, label string, dim, rail bool, action func()) {
	t := a.T
	row := ui.Row(c).Height(28).FillWidth().PaddingX(8).Gap(9).Radius(8).AlignItems(ui.Center)
	if rail {
		row.Justify(ui.Center)
	}
	color := t.Muted
	if dim {
		color = t.Dim
	}
	if row.Hovered() {
		row.Background(t.Hover)
		color = t.Foreground
	}
	row.Cursor(ui.CursorPointer)
	row.Children(func() {
		a.icon(c, icon, 14, color)
		if !rail {
			ui.Text(c, label).FontSize(13).TextColor(color)
		}
	})
	if row.Clicked() {
		action()
	}
}

// work is the tabs bar, the panes and the status bar.
func (a *App) work(c *ui.Context) {
	t := a.T
	ui.Column(c).Grow(1).FillHeight().Children(func() {
		a.tabsBar(c)
		ui.Row(c).Grow(1).FillWidth().Background(t.Canvas).Padding(8).Gap(8).Children(func() {
			ui.Box(c).Grow(1).FillHeight().Children(func() { a.paneArea(c) })
			if a.panel != "" {
				ui.Box(c).Width(panelWidth).FillHeight().Children(func() { a.panelView(c) })
			}
		})
		a.statusBar(c)
	})
}

// tabsBar is the row of tabs, the new tab button and the panel buttons.
func (a *App) tabsBar(c *ui.Context) {
	t := a.T
	ui.Column(c).Height(42).FillWidth().Children(func() {
		ui.Row(c).Grow(1).FillWidth().Background(t.Chrome).PaddingX(10).Gap(3).AlignItems(ui.Center).DragWindow().Children(func() {
			if !a.spaceEmpty() {
				a.tabList(c)
			} else {
				ui.Spacer(c)
			}
			a.newMenuButton(c)
			a.panelButton(c, "files", "Files")
			a.panelButton(c, "changes", "Changes")
		})
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
	})
}

// newMenuButton opens the new tab menu at the pointer.
func (a *App) newMenuButton(c *ui.Context) {
	t := a.T
	b := ui.Box(c).Size(26, 26).Radius(6).Center()
	if b.Hovered() {
		b.Background(t.Hover)
	}
	b.Cursor(ui.CursorPointer)
	b.Children(func() {
		color := t.Dim
		if b.Hovered() {
			color = t.Foreground
		}
		a.icon(c, "plus", 15, color)
	})
	if b.Clicked() {
		x, y, _ := b.PointerPosition()
		bounds := b.Bounds()
		a.menuX, a.menuY = bounds.X+x, bounds.Y+y
		a.OpenOverlay("new-menu")
	}
}

// tabList draws the tabs of the shown space, scrolled sideways.
func (a *App) tabList(c *ui.Context) {
	sp, active := a.S.Active()
	if sp == nil {
		return
	}
	ui.Row(c).Grow(1).FillHeight().Gap(3).AlignItems(ui.Center).Children(func() {
		for _, tab := range sp.Tabs {
			a.tab(c, tab, active != nil && tab.ID == active.ID)
		}
	})
}

// tab is one tab pill.
func (a *App) tab(c *ui.Context, tab *herdr.Tab, active bool) {
	t := a.T
	panes := a.S.Panes(tab.ID)
	name := herdr.TabName(tab, tab.Index)
	if strings.HasPrefix(name, "tb") && len(name) > 12 {
		name = "Terminal"
	}
	status := ""
	zoomed := false
	lead := herdr.PaneInfo{Program: herdr.Program(nil)}
	for _, p := range panes {
		if p.Status != "" {
			status = herdr.Worse(status, p.Status)
		}
		if p.Zoomed {
			zoomed = true
		}
		// The tab wears the program of its lead pane: the focused one, else the
		// first (the app's mods/tabs.js lead()).
		if lead.Terminal == "" || p.Focused {
			lead = p
		}
	}
	row := ui.Row(c).Height(29).PaddingX(10).Gap(7).Radius(7).AlignItems(ui.Center).Border(1, t.Chrome).Shrink(1)
	row.Background(t.Chrome)
	color := t.Muted
	if active {
		row.Background(t.Raised).Border(1, t.Border)
		color = t.Foreground
	} else if row.Hovered() {
		row.Background(t.Hover)
	}
	row.Cursor(ui.CursorPointer)
	row.Children(func() {
		a.programIcon(c, lead.Program, 14, t.Muted)
		label := name
		if zoomed {
			label += " Z"
		}
		ui.Text(c, label).FontSize(12.8).TextColor(color).MaxWidth(170).Ellipsis("…")
		switch status {
		case "working":
			a.icon(c, "spinner", 12, t.Working)
		case "blocked", "done":
			ui.Box(c).Size(6, 6).Radius(3).Background(t.Status(status))
		}
		if active {
			a.icon(c, "close", 12, t.Dim)
		}
	})
	if row.Clicked() {
		a.S.SelectTab(tab.ID)
	}
	if row.DoubleClicked() {
		a.Prompt("Rename tab", "Leave empty for its number", tab.Name, "tab.rename", tab.ID)
	}
	if row.RightClicked() {
		x, y, _ := row.PointerPosition()
		b := row.Bounds()
		id, spaceID := tab.ID, a.S.ActiveSpaceID()
		a.openMenuAt(b.X+x, b.Y+y, []MenuItem{
			{Label: "Rename", Icon: "square-pen", Do: func() { a.Prompt("Rename tab", "Leave empty for its number", tab.Name, "tab.rename", id) }},
			{Label: "New tab", Icon: "plus", Do: func() { a.newTab() }},
			{Label: "Close", Icon: "close", Do: func() { a.S.CloseTab(id) }},
		})
		_ = spaceID
	}
}

// panelButton is Files or Changes in the tabs bar.
func (a *App) panelButton(c *ui.Context, id, label string) {
	t := a.T
	on := a.panel == id
	row := ui.Row(c).Height(26).PaddingX(8).Gap(5).Radius(6).AlignItems(ui.Center).Background(t.Chrome)
	color := t.Muted
	if on {
		row.Background(t.Selected)
		color = t.Foreground
	}
	row.Cursor(ui.CursorPointer)
	if row.Hovered() && !on {
		row.Background(t.Hover)
	}
	row.Children(func() {
		a.icon(c, id, 14, color)
		ui.Text(c, label).FontSize(12).TextColor(color)
	})
	if row.Clicked() {
		a.OpenPanel(id)
	}
}

// spaceEmpty reports whether nothing is open at all.
func (a *App) spaceEmpty() bool {
	return a.S.ActiveSpaceID() == "" || len(a.S.Tabs()) == 0
}

// newTab adds a tab to the shown space, or a space when there is none.
func (a *App) newTab() {
	go func() {
		if a.spaceEmpty() {
			_, _, err := a.S.NewSpace("", "", "", nil)
			if err != nil {
				a.toast("Cannot start a terminal", err.Error(), "error")
			}
			return
		}
		if _, err := a.S.NewTab("", "", nil); err != nil {
			a.toast("Cannot start a terminal", err.Error(), "error")
		}
	}()
}

// paneArea draws the shown tab's panes as its tree arranges them.
func (a *App) paneArea(c *ui.Context) {
	sp, tab := a.S.Active()
	if sp == nil || tab == nil {
		a.emptyWork(c)
		return
	}
	panes := a.S.Panes(tab.ID)
	if len(panes) == 0 {
		a.emptyWork(c)
		return
	}
	if zoomed := a.S.Zoomed(); zoomed != "" {
		for _, p := range panes {
			if p.Pane == zoomed {
				a.paneCard(c, p, 10, 0)
				return
			}
		}
	}
	tree := a.S.Tree(tab.ID)
	if tree == nil {
		a.emptyWork(c)
		return
	}
	byPane := map[string]herdr.PaneInfo{}
	for _, p := range panes {
		byPane[p.Pane] = p
	}
	a.renderNode(c, tree, byPane, true, "row")
}

// renderNode draws a row, a column, or a pane. axis is the direction of the
// parent: a row's children grow across, a column's children grow down.
func (a *App) renderNode(c *ui.Context, n *herdr.Node, panes map[string]herdr.PaneInfo, root bool, axis string) {
	if n == nil {
		return
	}
	if n.Type == "row" || n.Type == "column" {
		var box ui.Element
		if n.Type == "row" {
			box = ui.Row(c)
		} else {
			box = ui.Column(c)
		}
		if root {
			box = box.Fill()
		} else {
			box = a.applySize(box, n, axis)
		}
		box = box.Gap(8)
		box.Children(func() {
			for _, child := range n.Children {
				a.renderNode(c, child, panes, false, n.Type)
			}
		})
		return
	}
	info, ok := panes[n.ID]
	if !ok {
		return
	}
	box := ui.Box(c)
	if root {
		box = box.Fill()
	} else {
		box = a.applySize(box, n, axis)
	}
	box.Children(func() { a.paneCard(c, info, 10, 0) })
}

// applySize gives a node its share along its parent: px, a weight, or equal.
func (a *App) applySize(el ui.Element, n *herdr.Node, axis string) ui.Element {
	if n.Size != nil {
		if n.Size.Weight != nil && *n.Size.Weight > 0 {
			if axis == "row" {
				return el.Grow(float32(*n.Size.Weight)).FillHeight()
			}
			return el.Grow(float32(*n.Size.Weight)).FillWidth()
		}
		if n.Size.PX != nil && *n.Size.PX > 0 {
			if axis == "row" {
				return el.Width(float32(*n.Size.PX)).FillHeight()
			}
			return el.Height(float32(*n.Size.PX)).FillWidth()
		}
	}
	if axis == "row" {
		return el.Grow(1).FillHeight()
	}
	return el.Grow(1).FillWidth()
}

// paneCard frames one pane: its header and its terminal.
func (a *App) paneCard(c *ui.Context, p herdr.PaneInfo, radius, pad float32) {
	t := a.T
	borderColor := t.Border
	if p.Focused {
		borderColor = t.Accent
	}
	card := ui.Column(c).Fill().Background(t.Background).Radius(radius).Border(1, borderColor).Clip()
	card.Children(func() {
		a.paneHeader(c, p)
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		ui.Box(c).Grow(1).FillWidth().Children(func() {
			a.terminalView(c, p)
		})
	})
}

// paneHeader is the pane's bar: program, folder, status, buttons.
func (a *App) paneHeader(c *ui.Context, p herdr.PaneInfo) {
	t := a.T
	row := ui.Row(c).Height(30).FillWidth().Background(t.Background).PaddingX(10).Gap(8).AlignItems(ui.Center)
	hovered := row.Hovered()
	row.Children(func() {
		if p.Terminal != "" {
			a.programIcon(c, p.Program, 14, t.Dim)
		}
		name := p.Title
		if name == "" {
			name = "shell"
		}
		ui.Text(c, name).FontSize(12).FontWeight(500).TextColor(t.Foreground).MaxWidth(260).Ellipsis("…")
		ui.Text(c, herdr.ShortFolder(p.Cwd)).FontSize(12).TextColor(t.Dim).Grow(1).Ellipsis("…")
		switch p.Status {
		case "working":
			a.icon(c, "spinner", 12, t.Working)
		case "blocked":
			ui.Box(c).Height(18).PaddingX(6).Radius(5).Background(Alpha(t.Blocked, 0.15)).Center().Children(func() {
				ui.Text(c, "Needs you").FontSize(10.5).FontWeight(600).TextColor(t.Blocked)
			})
		case "done":
			ui.Box(c).Size(7, 7).Radius(4).Background(t.Done)
		}
		if hovered {
			a.paneButton(c, "split-right", func() { a.split("right") })
			a.paneButton(c, "split-down", func() { a.split("down") })
			if p.Zoomed {
				a.paneButton(c, "unzoom", func() { a.S.ZoomPane(p.Pane) })
			} else {
				a.paneButton(c, "zoom", func() { a.S.ZoomPane(p.Pane) })
			}
			a.paneButton(c, "close", func() { a.S.ClosePane(p.Pane) })
		}
	})
	if row.Clicked() && !hovered {
		a.S.FocusPane(p.Pane)
	}
	if row.RightClicked() {
		x, y, _ := row.PointerPosition()
		b := row.Bounds()
		pane := p
		a.openMenuAt(b.X+x, b.Y+y, []MenuItem{
			{Label: "Split right", Icon: "split-right", Do: func() { a.splitPane(pane.Pane, "right") }},
			{Label: "Split down", Icon: "split-down", Do: func() { a.splitPane(pane.Pane, "down") }},
			{Label: "Zoom", Icon: "zoom", Do: func() { a.S.ZoomPane(pane.Pane) }},
			{Label: "Close pane", Icon: "close", Do: func() { a.S.ClosePane(pane.Pane) }},
		})
	}
}

func (a *App) paneButton(c *ui.Context, icon string, action func()) {
	t := a.T
	b := ui.Box(c).Padding(3).Radius(4)
	if b.Hovered() {
		b.Background(t.Hover)
	}
	b.Cursor(ui.CursorPointer)
	b.Children(func() {
		color := t.Dim
		if b.Hovered() {
			color = t.Foreground
		}
		a.icon(c, icon, 14, color)
	})
	if b.Clicked() {
		action()
	}
}

// splitPane focuses a pane and splits it.
func (a *App) splitPane(pane, direction string) {
	a.S.FocusPane(pane)
	a.split(direction)
}

func (a *App) split(direction string) {
	go func() {
		if _, err := a.S.Split(direction, nil); err != nil {
			a.toast("Cannot split the pane", err.Error(), "error")
		}
	}()
}

// terminalView draws the core terminal of a pane, attaching on first sight.
func (a *App) terminalView(c *ui.Context, p herdr.PaneInfo) {
	term := a.terminalFor(p)
	if term == nil {
		return
	}
	view := terminal.View(c, term).Fill()
	if a.focusPane == p.Pane {
		a.focusPane = ""
		view.AutoFocus()
	}
	if view.Clicked() {
		a.S.FocusPane(p.Pane)
	}
	if view.RightClicked() {
		x, y, _ := view.PointerPosition()
		b := view.Bounds()
		pane := p
		a.openMenuAt(b.X+x, b.Y+y, []MenuItem{
			{Label: "Split right", Icon: "split-right", Do: func() { a.splitPane(pane.Pane, "right") }},
			{Label: "Split down", Icon: "split-down", Do: func() { a.splitPane(pane.Pane, "down") }},
			{Label: "Zoom", Icon: "zoom", Do: func() { a.S.ZoomPane(pane.Pane) }},
			{Label: "Close pane", Icon: "close", Do: func() { a.S.ClosePane(pane.Pane) }},
		})
	}
}

// terminalFor returns the pane's terminal view, starting its attachment in
// the background the first time the pane shows.
func (a *App) terminalFor(p herdr.PaneInfo) *terminal.Terminal {
	if p.Terminal == "" {
		return nil
	}
	key := p.Pane + "|" + p.Terminal
	a.mu.Lock()
	if entry, ok := a.terms[key]; ok {
		a.mu.Unlock()
		return entry.term
	}
	if a.starting[key] {
		a.mu.Unlock()
		return nil
	}
	a.starting[key] = true
	a.mu.Unlock()

	go a.startTerminal(key, p)
	return nil
}

// startTerminal attaches to the pane's terminal and makes its view.
func (a *App) startTerminal(key string, p herdr.PaneInfo) {
	t := a.T
	conn, err := a.S.Attach(p.Terminal, 80, 24, t.TerminalForeground, t.Background, t.Cursor, 8, 16)
	if err != nil {
		a.mu.Lock()
		delete(a.starting, key)
		a.mu.Unlock()
		a.toast("Cannot show the terminal", err.Error(), "error")
		return
	}
	term, err := terminal.New(terminal.Options{
		Conn:         conn,
		Font:         terminal.Font{Family: TerminalFont, Size: 14},
		Theme:        terminalTheme(t),
		Padding:      terminal.Padding{X: 14, Y: 12},
		PaddingColor: terminal.PaddingExtend,
	})
	if err != nil {
		_ = conn.Close()
		a.mu.Lock()
		delete(a.starting, key)
		a.mu.Unlock()
		a.toast("Cannot start the terminal", err.Error(), "error")
		return
	}
	a.mu.Lock()
	a.terms[key] = &paneTerm{key: key, pane: p.Pane, addr: p.Terminal, term: term, conn: conn}
	delete(a.starting, key)
	a.mu.Unlock()
	a.Repaint()
	go func() {
		<-term.Done()
		a.S.ClosePane(p.Pane)
	}()
}

// collect detaches the views of panes that are no longer shown.
func (a *App) collect() {
	_, tab := a.S.Active()
	live := map[string]bool{}
	if tab != nil {
		for _, p := range a.S.Panes(tab.ID) {
			live[p.Pane+"|"+p.Terminal] = true
		}
	}
	a.mu.Lock()
	var gone []*paneTerm
	for key, entry := range a.terms {
		if !live[key] {
			gone = append(gone, entry)
			delete(a.terms, key)
		}
	}
	a.mu.Unlock()
	for _, entry := range gone {
		_ = entry.term.Close()
	}
}

// terminalTheme is the terminal's colors from the window's theme.
func terminalTheme(t *Theme) *terminal.Theme {
	th := &terminal.Theme{
		Foreground: t.TerminalForeground,
		Background: t.Background,
		Cursor:     t.Cursor,
		Selection:  t.Selection,
		Palette:    t.Ansi,
	}
	th.CursorText = t.Background
	return th
}

// statusBar is the bottom row: machine, branch, agents and the palette hint.
func (a *App) statusBar(c *ui.Context) {
	t := a.T
	ui.Column(c).Height(28).FillWidth().Children(func() {
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		row := ui.Row(c).Grow(1).FillWidth().Background(t.Chrome).PaddingX(14).Gap(16).AlignItems(ui.Center)
		row.Children(func() {
			icon, machine := "monitor", "This computer"
			if sp, _ := a.S.Active(); sp != nil && sp.Machine != nil {
				icon, machine = "server", *sp.Machine
			}
			a.statusItem(c, icon, machine)
			if b := a.S.Branch(); b != "" {
				a.statusItem(c, "branch", b)
			}
			agents := a.S.Agents()
			needs := 0
			for _, ag := range agents {
				if ag.Status == "blocked" {
					needs++
				}
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(6, 6).Radius(3).Background(t.Dim)
				ui.Text(c, fmt.Sprintf("%d agents", len(agents))).FontSize(11.8).TextColor(t.Muted)
				if needs > 0 {
					ui.Text(c, fmt.Sprintf("· %d needs you", needs)).FontSize(11.8).TextColor(t.Blocked)
				}
			})
			ui.Spacer(c)
			hint := ui.Text(c, paletteHint()).FontSize(11.8).TextColor(t.Dim)
			if hint.Clicked() {
				a.OpenOverlay("palette")
			}
		})
	})
}

func (a *App) statusItem(c *ui.Context, icon, label string) {
	t := a.T
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		a.icon(c, icon, 13, t.Muted)
		ui.Text(c, label).FontSize(11.8).TextColor(t.Muted)
	})
}

// emptyWork is what the work area says before anything is open.
func (a *App) emptyWork(c *ui.Context) {
	t := a.T
	ui.Box(c).Fill().Center().Children(func() {
		ui.Column(c).Gap(10).AlignItems(ui.Center).Children(func() {
			a.logo(c, 40)
			ui.Text(c, "No spaces yet").FontSize(15).FontWeight(600).TextColor(t.Foreground)
			ui.Text(c, "Press ⌘N for a new space, or use the sidebar.").FontSize(12.5).TextColor(t.Dim)
		})
	})
}

func isMac() bool { return osGOOS() == "darwin" }

var _ = time.Now

// Attach gives the window its world once the core is reached.
func (a *App) Attach(s *herdr.Session) {
	a.mu.Lock()
	a.S = s
	a.mu.Unlock()
	a.Repaint()
}

// Fail shows why the window cannot reach herdr-core.
func (a *App) Fail(err error) {
	a.err = err
	a.Repaint()
}

// paletteHint is the bottom right's live hint for the command palette.
func paletteHint() string {
	if k := prettyKeys(KeyFor("app.palette")); k != "" {
		return k + "  commands"
	}
	return "commands"
}

// resizeHandle drags the sidebar's edge, as Herdr's divider does.
func (a *App) resizeHandle(c *ui.Context) {
	t := a.T
	ui.Overlay(c, func() {
		handle := ui.Box(c).Absolute().Top(0).Bottom(0).Width(6).Left(a.SidebarWidth - 2).Cursor(ui.CursorResizeEW)
		if handle.Hovered() {
			handle.Background(Alpha(t.Border, 0.8))
		}
		if dx, _, ok := handle.Dragged(); ok {
			width := a.SidebarWidth + dx
			if width < 44 {
				width = 44
			}
			if width > 480 {
				width = 480
			}
			a.SidebarWidth = width
		}
	})
}
