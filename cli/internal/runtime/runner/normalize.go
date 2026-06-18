package runner

import (
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/runtime/plugin"
)

type metricEntry struct {
	name  string
	value float64
}

func (r *Runner) normalizeResponse(source string, resp *plugin.Response) ([]store.Item, []metricEntry, []store.Member) {
	var items []store.Item
	var metrics []metricEntry

	for _, e := range resp.Entries {
		mode := ""
		if e.Metadata != nil {
			if m, ok := e.Metadata["mode"].(string); ok {
				mode = m
			}
		}

		if mode == "metric" {
			value := 0.0
			if e.Metadata != nil {
				if v, ok := e.Metadata["metric_value"].(float64); ok {
					value = v
				}
			}
			metrics = append(metrics, metricEntry{name: e.Title, value: value})
			continue
		}

		member := r.store.Team.Resolve(e.Member)
		link := e.Link
		if link == "" && e.Metadata != nil {
			if url, ok := e.Metadata["web_url"].(string); ok {
				link = url
			}
		}
		fp := store.Fingerprint(source, link, e.Title, member)
		items = append(items, store.Item{
			Fingerprint: fp,
			Source:      source,
			Member:      member,
			Category:    e.Category,
			Title:       e.Title,
			Detail:      e.Detail,
			EntryDate:   e.EntryDate,
			Priority:    e.Priority,
			Link:        link,
		})
	}

	for _, met := range resp.Metrics {
		metrics = append(metrics, metricEntry{name: met.Name, value: met.Value})
	}

	members := make([]store.Member, 0, len(resp.TeamMembers))
	for _, t := range resp.TeamMembers {
		members = append(members, store.Member{
			Name:                t.Name,
			Email:               t.Email,
			Role:                t.Role,
			Department:          t.Department,
			Branch:              t.Branch,
			Registration:        t.Registration,
			ManagerRef:          t.ManagerRegistration,
			ManagerRegistration: t.ManagerRegistration,
			Source:              source,
		})
	}

	return items, metrics, members
}

func (r *Runner) partitionEntries(result SourceResult) ([]store.Item, []metricEntry) {
	var items []store.Item
	var metrics []metricEntry

	for _, e := range result.Entries {
		mode := ""
		if e.Metadata != nil {
			if m, ok := e.Metadata["mode"].(string); ok {
				mode = m
			}
		}

		if mode == "metric" {
			value := 0.0
			if e.Metadata != nil {
				if v, ok := e.Metadata["metric_value"].(float64); ok {
					value = v
				}
			}
			metrics = append(metrics, metricEntry{name: e.Title, value: value})
			continue
		}

		member := r.store.Team.Resolve(e.Member)
		link := ""
		if e.Metadata != nil {
			if url, ok := e.Metadata["web_url"].(string); ok {
				link = url
			}
		}
		fp := store.Fingerprint(e.Source, link, e.Title, member)
		items = append(items, store.Item{
			Fingerprint: fp,
			Source:      e.Source,
			Member:      member,
			Category:    e.Category,
			Title:       e.Title,
			Detail:      e.Detail,
			EntryDate:   e.EntryDate,
			Priority:    e.Priority,
			Link:        link,
		})
	}
	return items, metrics
}
