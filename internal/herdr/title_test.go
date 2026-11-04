package herdr

import (
	"encoding/json"
	"testing"
	"time"
)

// TestAgentTitleFollowsEvents checks the core event path through the cached
// agents list and the redraw notification, including agents that report their
// session title only through OSC 7501 status.
func TestAgentTitleFollowsEvents(t *testing.T) {
	s := sessionWithOnePane(`{"address":"t10","state":"live","title":"Old session","status":{"app":"pi","state":"working"}}`)
	s.activeTab = map[string]string{"sp1": "tb1"}
	s.git = map[string]gitInfo{}
	s.gitBusy = map[string]bool{}
	s.savedAt = time.Now()
	changes := 0
	s.OnChange = func() { changes++ }
	events := []struct {
		kind string
		data string
		want string
	}{
		{"title", `{"title":"New session"}`, "New session"},
		{"status", `{"status":{"app":"pi","state":"working","title":"Reported session"}}`, "Reported session"},
		{"title", `{"title":"Terminal fallback"}`, "Reported session"},
		{"status", `{"status":{"app":"pi","state":"working","title":"Renamed session"}}`, "Renamed session"},
		{"status", `{"status":{"app":"pi","state":"working"}}`, "Terminal fallback"},
		{"title", `{"title":""}`, ""},
	}
	for i, event := range events {
		line, err := json.Marshal(map[string]any{
			"method": "event",
			"params": map[string]any{"type": event.kind, "terminal": "t10", "data": json.RawMessage(event.data)},
		})
		if err != nil {
			t.Fatal(err)
		}
		s.handleLine(line)
		agents := s.Agents()
		if len(agents) != 1 || agents[0].Title != event.want {
			t.Fatalf("event %d: agents %+v, want title %q", i, agents, event.want)
		}
		if changes != i+1 {
			t.Fatalf("event %d: got %d redraws, want %d", i, changes, i+1)
		}
	}
	s.applySnapshot(json.RawMessage(`{"terminals":[{"address":"t10","state":"live","title":"Terminal snapshot","status":{"state":"working","title":"Reported snapshot"}}]}`))
	s.reindex()
	if got := s.Agents()[0].Title; got != "Reported snapshot" {
		t.Fatalf("snapshot title = %q, want reported title", got)
	}
}
