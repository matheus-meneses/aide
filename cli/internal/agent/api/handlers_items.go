package api

import "net/http"

func (h *handlers) handleItems(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	query := r.URL.Query().Get("q")

	var items interface{}
	var err error

	if query != "" {
		items, err = h.a.Store().Items.Search(query)
	} else {
		items, err = h.a.Store().Items.QueryOpen(source, "", "")
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *handlers) handleToday(w http.ResponseWriter, _ *http.Request) {
	events, err := h.a.Store().Items.TodayEvents()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *handlers) handleStatus(w http.ResponseWriter, _ *http.Request) {
	counts, _ := h.a.Store().Items.CountOpenBySource()
	health, _ := h.a.Store().Runs.AllHealth()
	metrics, _ := h.a.Store().Metrics.Latest("")
	events, _ := h.a.Store().Items.TodayEvents()

	if counts == nil {
		counts = map[string]int{}
	}

	status := map[string]interface{}{
		"counts":       counts,
		"health":       health,
		"metrics":      metrics,
		"today_events": len(events),
	}
	writeJSON(w, http.StatusOK, status)
}
