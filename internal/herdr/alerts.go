package herdr

// indexAgentAlertsLocked keeps transitions until the window consumes them,
// including a working/done pair that arrives between two frames. The initial
// snapshot establishes a baseline; navigation and repeated reports stay quiet.
func (s *Session) indexAgentAlertsLocked(next []*Agent) {
	statuses := make(map[string]string, len(next))
	shown, _ := s.activeLocked()
	for _, ag := range next {
		// Acknowledged completions display as idle in the sidebar. Alerts
		// follow the report itself, including later turns of a seen agent.
		status := ag.Status
		if term := s.terminals[ag.Address]; term != nil {
			status = term.Word()
		}
		statuses[ag.Address] = status
		if !s.alertsReady || status != "done" && status != "blocked" {
			continue
		}
		if s.alertStatuses[ag.Address] == status || shown != nil && shown.ID == ag.Space {
			continue
		}
		alert := *ag
		alert.Status = status
		s.agentAlerts = append(s.agentAlerts, alert)
	}
	s.alertStatuses = statuses
	s.agents, s.alertsReady = next, true
}

// TakeAgentAlerts drains new done/blocked transitions from agents outside the
// space shown when they reported. It is safe alongside the core event stream.
func (s *Session) TakeAgentAlerts() []Agent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.agentAlerts
	s.agentAlerts = nil
	return out
}
