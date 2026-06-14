package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type toolCall struct {
	Tool   string            `json:"tool"`
	Params map[string]string `json:"params"`
	Reason string            `json:"reason"`
}

func (a *Agent) think(ctx context.Context, state agentState, history []string) toolCall {
	prompt := a.buildAgentPrompt(state, history)

	messages := []ChatMessage{
		{Role: "user", Content: prompt},
	}

	llm := a.getLLM()
	resp, usage, err := llm.Chat(ctx, messages)
	if err != nil {
		alog.Error("LLM error: %v", err)
		return toolCall{Tool: "done", Reason: "LLM unreachable"}
	}

	if usage != nil {
		if err := a.store.Tokens.Record("agent", llm.Model(), usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens); err != nil {
			alog.Warn("failed to record token usage: %v", err)
		}
	}

	return parseToolCall(resp)
}

func parseToolCall(response string) toolCall {
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	start := strings.Index(response, "{")
	if start < 0 {
		alog.Warn("parse error: no JSON object found (response: %s)", response)
		return toolCall{Tool: "done", Reason: "parse error"}
	}

	dec := json.NewDecoder(strings.NewReader(response[start:]))
	var call toolCall
	if err := dec.Decode(&call); err != nil {
		alog.Warn("parse error: %v (response: %s)", err, response)
		return toolCall{Tool: "done", Reason: "parse error"}
	}

	if call.Params == nil {
		call.Params = make(map[string]string)
	}
	return call
}

func (a *Agent) executeTool(ctx context.Context, name string, params map[string]string) (string, error) {
	tool, ok := a.tools.Get(name)
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return tool.Execute(ctx, params)
}
