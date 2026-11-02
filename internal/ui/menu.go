package ui

import (
	"runtime"

	"github.com/egoist/mygo"
)

// InstallMenu puts Herdr's menu bar in place. MyGo's default macOS menu has
// File → Close Window on Cmd+W, which would take the shortcut before the
// window sees it and end the whole app; Herdr leaves the close keys to its
// own bindings, so Cmd+W closes the focused pane.
func InstallMenu() {
	if runtime.GOOS != "darwin" {
		return
	}
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	}))
}
