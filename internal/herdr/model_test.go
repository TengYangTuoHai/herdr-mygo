package herdr

import "testing"

func TestSpaceNameHome(t *testing.T) {
	rohan := "rohan"
	sp := &Space{ID: "rohan/sp1", Machine: &rohan, Folder: "/home/rohan"}
	if got := spaceNameWith(sp, nil, "/home/rohan"); got != "~" {
		t.Fatalf("home with no slash: %q", got)
	}
	sp.Folder = "/home/rohan/"
	if got := spaceNameWith(sp, nil, "/home/rohan/"); got != "~" {
		t.Fatalf("home with slashes: %q", got)
	}
	sp.Folder = "/srv/app"
	if got := spaceNameWith(sp, nil, "/home/rohan"); got != "app" {
		t.Fatalf("project folder: %q", got)
	}
	sp.Folder = "/"
	if got := spaceNameWith(sp, nil, "/home/rohan"); got != "/" {
		t.Fatalf("root: %q", got)
	}
	sp.Folder = ""
	if got := spaceNameWith(sp, nil, "/home/rohan"); got != "~" {
		t.Fatalf("no folder: %q", got)
	}
	sp.Name = "spike"
	if got := spaceNameWith(sp, nil, "/home/rohan"); got != "spike" {
		t.Fatalf("named space: %q", got)
	}
}

func TestTerminalMachine(t *testing.T) {
	cases := []struct{ addr, want string }{
		{"t7", ""},
		{"", ""},
		{"vps/t7", "vps"},
		{"4090/t12", "4090"},
		{"/t7", ""},
	}
	for _, c := range cases {
		if got := TerminalMachine(c.addr); got != c.want {
			t.Errorf("TerminalMachine(%q) = %q, want %q", c.addr, got, c.want)
		}
	}
}
