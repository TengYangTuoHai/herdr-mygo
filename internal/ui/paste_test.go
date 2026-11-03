package ui

import (
	"testing"

	"github.com/egoist/mygo/transfer"
)

func TestImageFormat(t *testing.T) {
	cases := []struct {
		name    string
		offered []transfer.Format
		want    transfer.Format
		ok      bool
	}{
		{"text alone", []transfer.Format{transfer.Text, transfer.HTML}, "", false},
		{"a screenshot", []transfer.Format{transfer.Text, transfer.PNG}, transfer.PNG, true},
		{"an image with its text", []transfer.Format{transfer.PNG, transfer.Text}, transfer.PNG, true},
		{"PNG before another image type", []transfer.Format{"image/tiff", transfer.PNG}, transfer.PNG, true},
		{"a JPEG", []transfer.Format{"image/jpeg", transfer.Text}, "image/jpeg", true},
		{"another image type", []transfer.Format{"image/svg+xml", transfer.Text}, "image/svg+xml", true},
		{"files alone", []transfer.Format{transfer.FileList, transfer.URIList}, "", false},
	}
	for _, c := range cases {
		got, ok := imageFormat(c.offered)
		if got != c.want || ok != c.ok {
			t.Errorf("%s: imageFormat(%v) = %q, %v, want %q, %v", c.name, c.offered, got, ok, c.want, c.ok)
		}
	}
}

func TestImageName(t *testing.T) {
	cases := []struct {
		format transfer.Format
		want   string
	}{
		{transfer.PNG, "pasted-image.png"},
		{"image/jpeg", "pasted-image.jpg"},
		{"image/gif", "pasted-image.gif"},
		{"image/webp", "pasted-image.webp"},
		{"image/svg+xml", "pasted-image.svg"},
		{"image/x-my-image", "pasted-image.xmyimage"},
		{"image/", "pasted-image"},
	}
	for _, c := range cases {
		if got := imageName(c.format); got != c.want {
			t.Errorf("imageName(%q) = %q, want %q", c.format, got, c.want)
		}
	}
}

// A paste for a terminal here is left to the terminal: its program reads
// this computer's clipboard itself, so the app pastes nothing of its own.
func TestPasteImageLeavesATerminalHere(t *testing.T) {
	app := &App{}
	d := transfer.New(transfer.NewItem(transfer.Bytes(transfer.PNG, []byte("PNG"))))
	if app.pasteImage(nil, "t7", d) {
		t.Error("a paste for a terminal here was taken")
	}
}
