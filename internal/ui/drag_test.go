package ui

import "testing"

func TestDropIndexAt(t *testing.T) {
	// Rows are 26 tall: the top half drops before the target, the bottom
	// half after it, and a target below the dragged row moves up a place.
	cases := []struct {
		name     string
		y        float32
		from, to int
		want     int
	}{
		{"top half, moving up", 5, 3, 1, 1},
		{"bottom half, moving up", 21, 3, 1, 2},
		{"top half, moving down", 5, 1, 3, 2},
		{"bottom half, moving down", 21, 1, 3, 3},
		{"just above the middle", 12, 1, 2, 1}, // already there: no move
		{"just below the middle", 14, 1, 2, 2},
	}
	for _, c := range cases {
		if got := dropIndexAt(c.y, 26, c.from, c.to); got != c.want {
			t.Errorf("%s: dropIndexAt(%v, %d, %d) = %d, want %d", c.name, c.y, c.from, c.to, got, c.want)
		}
	}
}
