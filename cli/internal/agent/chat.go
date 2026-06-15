package agent

import (
	"aide/cli/internal/agent/llm"
	"context"
	"errors"
	"fmt"
	"time"
)

// ChatRequest is a transport-free chat turn request.
type ChatRequest struct {
	Message   string
	SessionID string
}

var (
	// ErrChatEmptyMessage is returned when a chat turn has no message.
	ErrChatEmptyMessage = errors.New("message required")
	// ErrLLMNotConfigured is returned when no model is configured yet.
	ErrLLMNotConfigured = errors.New("llm not configured")
)

// StreamChat runs one chat turn end to end: it builds the system context,
// maintains session history, persists messages, records token usage and streams
// assistant tokens through emit. All HTTP/SSE concerns live in the api package;
// the core stays transport-free.
func (a *Agent) StreamChat(ctx context.Context, req ChatRequest, emit func(token string)) error {
	if req.Message == "" {
		return ErrChatEmptyMessage
	}
	if req.SessionID == "" {
		req.SessionID = "default"
	}

	sess := a.sessions.getOrCreate(req.SessionID)
	sess.mu.Lock()
	defer sess.mu.Unlock()

	sysCtx, err := BuildContext(a.store, a.clock.Now())
	if err != nil {
		return fmt.Errorf("building context: %w", err)
	}
	systemMsg := llm.ChatMessage{Role: "system", Content: sysCtx}

	if len(sess.history) == 0 {
		sess.history = []llm.ChatMessage{systemMsg}
		if persisted, err := a.store.Chat.LoadMessages(req.SessionID); err == nil {
			for _, m := range persisted {
				if m.Role == "user" || m.Role == "assistant" {
					sess.history = append(sess.history, llm.ChatMessage{Role: m.Role, Content: m.Content})
				}
			}
		}
	} else {
		sess.history[0] = systemMsg
	}

	sess.history = append(sess.history, llm.ChatMessage{Role: "user", Content: req.Message})
	sess.history = TrimHistory(sess.history, 30000)

	if cfg := a.getConfig(); cfg.Agent.LLMModel == "" || cfg.Agent.LLMURL == "" {
		sess.history = sess.history[:len(sess.history)-1]
		return ErrLLMNotConfigured
	}

	now := a.clock.Now().UTC().Format(time.RFC3339)
	if err := a.store.Chat.InsertMessage(req.SessionID, "user", req.Message, now); err != nil {
		alog.Warn("failed to persist user message: %v", err)
	}

	client := a.getLLM()
	full, usage, err := client.ChatStream(ctx, sess.history, emit)
	if err != nil {
		sess.history = sess.history[:len(sess.history)-1]
		return err
	}

	if usage != nil {
		if err := a.store.Tokens.Record("chat", client.Model(), usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens); err != nil {
			alog.Warn("failed to record token usage: %v", err)
		}
	}

	sess.history = append(sess.history, llm.ChatMessage{Role: "assistant", Content: full})

	if err := a.store.Chat.InsertMessage(req.SessionID, "assistant", full, a.clock.Now().UTC().Format(time.RFC3339)); err != nil {
		alog.Warn("failed to persist assistant message: %v", err)
	}

	return nil
}
