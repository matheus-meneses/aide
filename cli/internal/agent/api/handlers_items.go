package api

import (
	"aide/cli/internal/persistence/store"
	"net/http"
	"strings"
	"time"
)

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

func (h *handlers) handleNextEvent(w http.ResponseWriter, _ *http.Request) {
	next, err := h.a.Store().Items.NextEvent()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if next == nil {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	writeJSON(w, http.StatusOK, eventPayload(*next))
}

func (h *handlers) handleUpcomingEvents(w http.ResponseWriter, _ *http.Request) {
	events, err := h.a.Store().Items.UpcomingEventInfos()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	const maxEvents = 12
	if len(events) > maxEvents {
		events = events[:maxEvents]
	}
	out := make([]map[string]any, 0, len(events))
	for _, ev := range events {
		out = append(out, eventPayload(ev))
	}
	writeJSON(w, http.StatusOK, out)
}

func eventPayload(ev store.NextEventInfo) map[string]any {
	return map[string]any{
		"title":         strings.TrimPrefix(ev.Item.Title, "Meeting: "),
		"member":        ev.Item.Member,
		"time":          ev.Start.Format("15:04"),
		"start":         ev.Start.Format(time.RFC3339),
		"minutes_until": ev.MinutesUntil,
		"in_progress":   ev.InProgress,
	}
}

func (h *handlers) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.a.StatusSnapshot())
}
