# herdr-mygo

Herdr's desktop UI, rebuilt in Go with [MyGo](https://github.com/egoist/mygo)'s
native UI. The **core is Herdr's own**: [herdr-core](https://github.com/herdrdev/herdr-core)
(`hc`) still runs every terminal, every agent and the spaces/tabs document.
This app is only the window.

![Herdr's main window](docs/screenshot-main.png)

```
hc (herdr-core)                  this app
┌──────────────────────┐        ┌──────────────────────────────┐
│ terminals, PTYs      │ framed │ MyGo native UI (Go, GPU)      │
│ spaces/tabs doc      │◀──────▶│ sidebar · tabs · panes        │
│ agents' status       │  RPC   │ palette · settings · panels   │
└──────────────────────┘        └──────────────────────────────┘
```

## Run

Needs Go 1.27+ and `hc` on `PATH` (the Herdr app installs it in `~/.local/bin`).

```sh
go run .
```

`herdr-core` keeps running when the window closes, exactly as with the
original app: close the window, your agents keep working.

To build an app bundle:

```sh
go tool mygo build
```

The first run downloads MyGo's `libghostty-vt` (the terminal emulator library
the terminal plugin uses, the same Ghostty VT the original app uses).

## What is here

| Files | What it does |
|---|---|
| `internal/core` | `hc connect`, the framed JSON-RPC protocol: calls, event streams, terminal attach streams, flow control |
| `internal/herdr` | the world: the `herdr-app` document (spaces → tabs → pane trees), terminal records and events, commands, git, the program table, and the attach connection that makes a core terminal a MyGo terminal |
| `internal/ui` | the window: theme, frame, sidebar, tabs, pane cards, status bar, command palette, picker, new-tab menu, context menus, settings, welcome, Files and Changes panels, toasts |
| `internal/assets` | the original app's icons, fonts (JetBrains Mono, Geist) and logo, under their licenses |

The window follows the shipped `main` layout: a 268 px sidebar with the
spaces list, the agents list and the footer; a 42 px tab bar; pane cards
(radius 10, 1 px border, accent border while focused, 30 px header); a
28 px status bar; Files/Changes as a 368 px side panel. The default colors come from
Herdr's `graphite` theme, resolved with the same formula Herdr uses
(`settings.rs`), and terminals use the same palette, font and padding.
Settings → Look also offers Kero's tinted themes and gradient backgrounds,
with separate choices for light and dark windows, plus searchable Ghostty
themes (including installed files in Ghostty's themes folder). The chrome
and every open terminal follow the chosen palette. Choose “Theme” under
Accent to use the palette's accent, or pick an override.

Keys are read from `~/.config/herdr-mygo/keys.json` (or `$XDG_CONFIG_HOME`,
or `$HERDR_MYGO_HOME`, else the shipped `keys.macos.json` / `keys.json`), so
`cmd-shift-p` opens the palette,
`cmd-d` splits, `cmd-1`… select tabs, and so on. Terminals get every other
key, through Ghostty's key protocol handling in MyGo's terminal plugin.

## What works

- Spaces, tabs and splits stored in the core's shared document, with labels
  (`herdr.space`, `herdr.tab`) so terminals started outside the app join them
- **SSH machines**: add one from `~/.ssh/config` or by typing how ssh reaches
  it (Add machine in the sidebar footer, or Settings → Machines); each machine
  keeps its own copy of the document (`doc.get/set … on`), its spaces are
  grouped under its header in the sidebar, and its terminals run, attach,
  split, resize and report agents through the same core, addressed
  `machine/t3`. Remote spaces are first class: new space from the machine's
  `+`, tabs and splits there, a name of `~` from that machine's own home,
  Files listing and previewing that machine's folders (`machine.exec` + `sh`),
  and Changes running git there. A machine that needs herdr-core copied, a
  password or a host key opens `hc machine add` in a terminal inside the
  dialog. Settings →
  Machines lists each machine with its state, and Connect / Remove. SSH
  keepalives and connection reuse are always managed by herdr-core; the app
  has no keepalive switch
- Terminal attach: live output, typing, resize, colors reported to programs,
  selection, copy/paste, links, scrollback (MyGo terminal plugin)
- Find in terminal output and scrollback: ⌘F (Ctrl+Shift+F elsewhere),
  or “Find in terminal” in the palette or pane's right-click menu. Matches
  highlight as you type; Enter / Shift+Enter and the arrow buttons move
  toward older / newer output, with a match count. ⌘G / ⌘⇧G (F3 / Shift+F3)
  also navigate and reopen the last query. Escape closes the search and
  returns typing to the terminal. Peeked sub-agents can be searched too
- Focus and zoom per tab, click to focus, accent focus border
- Space/tab/pane commands: new, close, rename, move, select, next/prev,
  split right/down, close, zoom, focus/swap by direction. Closing a pane,
  tab or space asks for confirmation when it would end a running program;
  idle shells and finished terminals close immediately
- The agents list from terminals that report status (OSC 7501), by priority,
  each agent with its own logo: Herdr's program table's icons (Claude Code,
  Codex, Pi, OpenCode, Grok, Gemini, …), full color where the mark is, and
  DeepSeek's for `dsh-tui`/`dsh`; a row from another machine names it, and
  the workers an agent started (`hc run --bg …` from its terminal) hang under
  its row with the lines that lead to them, and a click on one looks at it
  over the tab: the bar over the terminal keeps it as a pane of the shown
  tab, or closes the look
- Command palette with fuzzy search, group headers and key chips
- Go-to-space-or-tab picker, new-tab menu, right-click menus on spaces,
  tabs and panes, prompt dialogs, toasts
- Settings for Look, Terminal, Machines and Alerts, with
  search (⌘F), theme previews and live controls; welcome and the keys list.
  Appearance follows the desktop, or stays light or dark. Light and dark
  choices persist separately in `light_theme` and `dark_theme`; older
  single-`theme` settings retain their previous appearance fallback.
  Graphite, Ink, Moss and Paper remain available alongside Kero's Petal,
  Mist, Meadow, Dusk, Tide, Apricot, Aurora, Dune, Neutral and Plain.
  Neutral is Kero's paper wash, named separately to keep Herdr's Paper palette.
  Look's font controls list bundled and installed families (monospaced families
  for terminals), with editable sizes from 8 to 40. Interface type updates
  across the window; terminal type updates open panes and peek without
  reconnecting. Font choices, sizes and window zoom persist in `settings.json`
- Alerts follow agent status reports: Done and Needs you send system notifications
  for agents in another space, with optional sound. On macOS, run
  `go tool mygo dev` or the app from `go tool mygo build` for notifications;
  the first alert asks for notification permission. Repeated reports and the
  initial session snapshot stay quiet. Glass and Ping are built-in chimes;
  either event can use a custom mono/stereo WAV file (up to 10 seconds and
  10 MB), previewed from Settings. By agent mutes sound for a program while
  keeping its system notifications. Sound choices and per-agent mutes persist in
  `settings.json`; disabling Play a sound keeps notifications enabled.
- The app keeps its own files, apart from herdr-app's, under
  `~/.config/herdr-mygo`: `settings.json` written as a setting changes, the
  keys, and the screen's view
- `herdr-app do` reaches the window: it answers on
  `$HERDR_APP_HOME/herdr-app.sock` (else `~/.config/herdr-app`), where that
  command looks, one JSON line each way. `tab.new` starts a tab and
  `terminal.keep` gives a terminal that is already running — a worker the
  core started — a pane, so it lands in the window
- Files tree with file preview; Changes with git status, diff and commit
- The sidebar collapses to nothing — ⌘B, or the button in the brand bar —
  easing closed as MyGo's sidebar example does (`Animate` + `ClipX`), and
  opens again at the width it had; the width and whether it is open are
  kept in `settings.json`
- Sidebar drag to resize, spaces and machine groups dragged into order,
  panes resized by dragging the divider between them, window drag on the
  brand and tab bars, traffic lights where Herdr puts them

## What is not here yet

The original app is a general layout/mod engine: layouts are JSON, panels
are JavaScript mods. This port draws Herdr's shipped interface in Go, so:

- No `layouts/*.json` or `mods/*` execution: `terminals` and `win98` layouts,
  and user mods, do not load. Their look is fixed to `main`.
- No worktrees, terminal peek or floating blocks; panes can be split,
  focused and resized by their divider, but not dragged to move; tabs
  cannot be drag-reordered yet.
- Machines are added over ssh only (no `--via` commands or temporary
  machines); Files reads remote folders one level at a time through
  `machine.exec`, so it has no git-ignore dimming and refreshes every few
  seconds rather than on every change.
- No session settling, update checks or telemetry, and only `tab.new` and
  `terminal.keep` answer `herdr-app do`: `ls`, `status`, `shot` and the rest
  of its commands are not here yet. Previews (Files, Changes) are read-only
  except the commit box.
- Option as Alt is hidden while MyGo v0.7.4 lacks runtime and left/right
  Option control. Existing saved values are retained for future support.
- Updates and Privacy are hidden from Settings and its search until an updater
  and telemetry exist. Existing saved values are retained for future support.

## How it is checked

`internal/ui/shot_test.go` renders the window without a window and saves a
2× PNG, which is compared against the original's own `herdr-app shot` at the
same size and scale. The screenshots in `docs/` were made that way.

```sh
SHOT_OUT=/tmp/main.png go test ./internal/ui -run TestShot
SHOT_OVERLAY=palette SHOT_OUT=/tmp/palette.png go test ./internal/ui -run TestShot
SHOT_PANEL=files SHOT_OUT=/tmp/files.png go test ./internal/ui -run TestShot
SHOT_ACTION=pane.split:down SHOT_OUT=/tmp/split.png go test ./internal/ui -run TestShot
```

## Research

[docs/herdr-ui-spec.md](docs/herdr-ui-spec.md) is the study of the original
app this port is built from: the model, the layout metrics, the theme, every
panel and overlay, keys and commands, and where each piece lives in Go.
