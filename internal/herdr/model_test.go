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

// TestSetWeightsKeepsTheDragsSizes moves the divider between two panes: the
// tree takes the new weights, the document writes them, and reading it back
// gives the same sizes. The panes and tabs a window does not show are left
// alone.
func TestSetWeightsKeepsTheDragsSizes(t *testing.T) {
	t.Setenv("HERDR_MYGO_HOME", t.TempDir())
	half := 0.5
	one := 1.0
	tab := &Tab{ID: "tb1", Root: &Node{ID: "tb1", Type: "row", Children: []*Node{
		{ID: "p1", Type: "block", Size: &Size{Weight: &half}},
		{ID: "p2", Type: "block", Size: &Size{Weight: &half}},
	}}}
	other := &Tab{ID: "tb2", Root: &Node{ID: "tb2", Type: "row", Children: []*Node{
		{ID: "p3", Type: "block", Size: &Size{Weight: &one}},
	}}}
	sp := &Space{ID: "sp1", Tabs: []*Tab{tab, other}}
	s := &Session{
		spaces:    []*Space{sp},
		active:    sp.ID,
		activeTab: map[string]string{sp.ID: tab.ID},
		tabs:      map[string]*tabView{tab.ID: {focused: "p1"}, other.ID: {focused: "p3"}},
		terminals: map[string]*Terminal{},
	}

	if !s.SetWeights(tab.ID, map[string]float64{"p1": 0.7, "p2": 0.3}) {
		t.Fatal("SetWeights found no tab")
	}
	got := s.Tree(tab.ID)
	for _, c := range []struct {
		pane string
		want float64
	}{{"p1", 0.7}, {"p2", 0.3}} {
		n := got.find(c.pane)
		if n == nil || n.Size == nil || n.Size.Weight == nil {
			t.Fatalf("%s lost its size: %+v", c.pane, n)
		}
		if w := *n.Size.Weight; w < c.want-1e-9 || w > c.want+1e-9 {
			t.Fatalf("%s weight = %v, want %v", c.pane, w, c.want)
		}
	}

	// What the document keeps is what a drag left.
	back := fromDoc(captured(s.spaces, nil), nil)
	if back[0].Tabs[0].Root.find("p1").Size.Weight == nil || *back[0].Tabs[0].Root.find("p1").Size.Weight != 0.7 {
		t.Fatalf("the document lost the drag: %+v", back[0].Tabs[0].Root.Children[0].Size)
	}

	// A pane of another tab, and a tab that is not here, are left alone, and
	// the other tab keeps its own size.
	if !s.SetWeights(tab.ID, map[string]float64{"p3": 0.2}) {
		t.Fatal("SetWeights lost the tab it was given")
	}
	if w := *other.Root.find("p3").Size.Weight; w != 1 {
		t.Fatalf("another tab's pane moved to %v", w)
	}
	if s.SetWeights("not-a-tab", map[string]float64{"p1": 0.1}) {
		t.Fatal("SetWeights found a tab that is not there")
	}
}
