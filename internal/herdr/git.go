package herdr

// Worse returns the more urgent of two status words.
func Worse(a, b string) string { return worse(a, b) }

// GitStat is what the sidebar says about a place's repo.
type GitStat struct {
	Repo   string
	Branch string
	Ahead  int
	Behind int
}

// termMap copies the terminal records, for naming a space.
func (s *Session) termMap() map[string]*Terminal { return s.termSnapshot() }

// TermMap is the terminal records by address.
func (s *Session) TermMap() map[string]*Terminal { return s.termSnapshot() }

// GitOf returns the git state of the folder a space's first terminal is in.
func (s *Session) GitOf(sp *Space) GitStat {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := sp.Folder
	for _, t := range sp.Tabs {
		for _, b := range t.Root.blocks() {
			term := s.terminals[b.terminal()]
			if term == nil {
				continue
			}
			if term.Foreground != nil && term.Foreground.Cwd != "" {
				dir = term.Foreground.Cwd
			} else if term.Cwd != "" {
				dir = term.Cwd
			}
			goto found
		}
	}
found:
	if g, ok := s.git[gitKey(machineOf(sp), dir)]; ok {
		return GitStat{Repo: g.Repo, Branch: g.Branch, Ahead: g.Ahead, Behind: g.Behind}
	}
	return GitStat{}
}
