package agent

import "net/http"

func (a *Agent) handleItems(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	query := r.URL.Query().Get("q")

	var items interface{}
	var err error

	if query != "" {
		items, err = a.store.Items.Search(query)
	} else {
		items, err = a.store.Items.QueryOpen(source, "", "")
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *Agent) handleToday(w http.ResponseWriter, _ *http.Request) {
	events, err := a.store.Items.TodayEvents()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (a *Agent) handleStatus(w http.ResponseWriter, _ *http.Request) {
	counts, _ := a.store.Items.CountOpenBySource()
	health, _ := a.store.Runs.AllHealth()
	metrics, _ := a.store.Metrics.Latest("")
	events, _ := a.store.Items.TodayEvents()

	status := map[string]interface{}{
		"counts":       counts,
		"health":       health,
		"metrics":      metrics,
		"today_events": len(events),
	}
	writeJSON(w, http.StatusOK, status)
}
