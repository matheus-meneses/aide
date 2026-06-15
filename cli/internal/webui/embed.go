package webui

import "embed"

//go:embed all:frontend/dist
var frontendFS embed.FS
