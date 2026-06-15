package render

import (
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/plugin"
	"context"
	"fmt"
)

func sourcePluginMap(cfg *config.Config) map[string]string {
	m := make(map[string]string, len(cfg.Sources))
	for name, src := range cfg.Sources {
		if src.Plugin != "" {
			m[name] = src.Plugin
		}
	}
	return m
}

type sourcePlugin interface {
	Classify(item store.Item) string
	RenderSection(heading string, items []store.Item)
}

func pluginFor(sourceName string, cfg map[string]string) sourcePlugin {
	pluginName := cfg[sourceName]
	if pluginName == "" {
		pluginName = sourceName
	}
	mgr := plugin.NewManager()
	m, err := mgr.Get(pluginName)
	if err != nil {
		return &defaultPlugin{}
	}
	if m.Render.Custom {
		return &pluginRenderer{manifest: m}
	}
	return &defaultPlugin{}
}

type pluginRenderer struct {
	manifest *plugin.Manifest
}

func (p *pluginRenderer) Classify(item store.Item) string {
	if item.Category != "" {
		return item.Category
	}
	return "Items"
}

func (p *pluginRenderer) RenderSection(heading string, items []store.Item) {
	entries := make([]plugin.Entry, 0, len(items))
	for _, item := range items {
		entries = append(entries, plugin.Entry{
			Member:    item.Member,
			Category:  item.Category,
			Title:     item.Title,
			Detail:    item.Detail,
			EntryDate: item.EntryDate,
			Priority:  item.Priority,
			Link:      item.Link,
		})
	}

	req := &plugin.Request{
		Action:  "render",
		Heading: heading,
		Items:   entries,
	}

	resp, _, err := plugin.Execute(context.Background(), p.manifest, req)
	if err != nil {
		fmt.Printf("  [!] plugin render failed (%s): %v\n", p.manifest.Name, err)
		(&defaultPlugin{}).RenderSection(heading, items)
		return
	}

	for _, line := range resp.Lines {
		fprintf("%s\n", line)
	}
}
