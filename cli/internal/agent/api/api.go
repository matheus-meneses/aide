package api

import (
	"aide/cli/internal/agent"
)

type handlers struct {
	a *agent.Agent
}

const maxChatBodyBytes = 1 << 20
