package api

import (
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
	title := strings.TrimPrefix(next.Item.Title, "Meeting: ")
	writeJSON(w, http.StatusOK, map[string]any{
		"title":         title,
		"member":        next.Item.Member,
		"time":          next.Start.Format("15:04"),
		"start":         next.Start.Format(time.RFC3339),
		"minutes_until": next.MinutesUntil,
		"in_progress":   next.InProgress,
	})
}

func (h *handlers) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.a.StatusSnapshot())
}
