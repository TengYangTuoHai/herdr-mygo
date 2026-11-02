// Package assets carries the pictures, icons and fonts the window draws:
// Herdr's own, under the licenses its assets ship with.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed files
var root embed.FS

// Files holds the assets, under files/.
var Files, _ = fs.Sub(root, "files")
