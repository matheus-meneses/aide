package render

import "aide/cli/internal/store"

type sourcePlugin interface {
	Classify(item store.Item) string
	RenderSection(heading string, items []store.Item)
}

var plugins = map[string]sourcePlugin{}

func registerSource(name string, p sourcePlugin) {
	plugins[name] = p
}

func pluginFor(source string) sourcePlugin {
	if p, ok := plugins[source]; ok {
		return p
	}
	return &defaultPlugin{}
}
