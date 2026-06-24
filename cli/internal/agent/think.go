package agent

import (
	"aide/cli/internal/agent/llm"
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

func (a *Agent) think(ctx context.Context, messages []llm.ChatMessage, tools []llm.ToolDefinition) (*llm.ChatResult, error) {
	client := a.getLLM()
	result, err := client.ChatWithTools(ctx, messages, tools)
	if err != nil {
		return nil, err
	}

	if result.Usage != nil {
		if err := a.store.Tokens.Record("agent", client.Model(), result.Usage.PromptTokens, result.Usage.CompletionTokens, result.Usage.TotalTokens); err != nil {
			alog.Warn("failed to record token usage: %v", err)
		}
	}

	return result, nil
}

func (a *Agent) toolDefinitions() []llm.ToolDefinition {
	defs := a.tools.Definitions()
	out := make([]llm.ToolDefinition, 0, len(defs))
	for _, d := range defs {
		out = append(out, llm.ToolDefinition{
			Name:        d.Name,
			Description: d.Description,
			Parameters:  d.InputSchema,
		})
	}
	return out
}

func (a *Agent) executeTool(ctx context.Context, name string, params map[string]string) (string, error) {
	tool, ok := a.tools.Get(name)
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return tool.Execute(ctx, params)
}

func argsToParams(raw json.RawMessage) map[string]string {
	params := make(map[string]string)
	if len(raw) == 0 {
		return params
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return params
	}
	for k, v := range m {
		switch vv := v.(type) {
		case nil:
			continue
		case string:
			params[k] = vv
		default:
			b, _ := json.Marshal(vv)
			params[k] = string(b)
		}
	}
	return params
}

func argString(raw json.RawMessage, key string) string {
	return argsToParams(raw)[key]
}

func fallbackToolCall(content string) (llm.ToolCall, bool) {
	if !strings.Contains(content, "{") {
		return llm.ToolCall{}, false
	}
	call := parseToolCall(content)
	if call.Tool == "" || (call.Tool == "done" && call.Reason == "parse error") {
		return llm.ToolCall{}, false
	}
	args, err := json.Marshal(call.Params)
	if err != nil {
		return llm.ToolCall{}, false
	}
	return llm.ToolCall{Name: call.Tool, Arguments: args}, true
}

func parseToolCall(response string) toolCall {
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	start := strings.Index(response, "{")
	if start < 0 {
		return toolCall{Tool: "done", Reason: "parse error"}
	}

	dec := json.NewDecoder(strings.NewReader(response[start:]))
	var call toolCall
	if err := dec.Decode(&call); err != nil {
		return toolCall{Tool: "done", Reason: "parse error"}
	}

	if call.Params == nil {
		call.Params = make(map[string]string)
	}
	return call
}
