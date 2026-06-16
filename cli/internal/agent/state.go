package agent

// StatusSnapshot returns the agent's current operational state: open-item counts
// by source, per-source health, the latest metrics and today's event count. It
// is the single source for both the `/status` slash command and the HTTP
// `GET /api/status` handler, keeping the two query paths in sync.
func (a *Agent) StatusSnapshot() map[string]any {
	counts, _ := a.store.Items.CountOpenBySource()
	if counts == nil {
		counts = map[string]int{}
	}
	health, _ := a.store.Runs.AllHealth()
	metrics, _ := a.store.Metrics.Latest("")
	events, _ := a.store.Items.TodayEvents()

	return map[string]any{
		"counts":       counts,
		"health":       health,
		"metrics":      metrics,
		"today_events": len(events),
	}
}
