package agent

import (
	"aide/cli/internal/agent/llm"
	"context"
	"sync"
	"time"
)

type chatSession struct {
	history    []llm.ChatMessage
	mu         sync.Mutex
	lastAccess time.Time
}

type sessionManager struct {
	mu       sync.Mutex
	sessions map[string]*chatSession
	ttl      time.Duration
}

func newSessionManager(ttl time.Duration) *sessionManager {
	return &sessionManager{
		sessions: make(map[string]*chatSession),
		ttl:      ttl,
	}
}

func (m *sessionManager) getOrCreate(id string) *chatSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[id]; ok {
		s.lastAccess = time.Now()
		return s
	}
	s := &chatSession{lastAccess: time.Now()}
	m.sessions[id] = s
	return s
}

func (m *sessionManager) evictExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff := time.Now().Add(-m.ttl)
	for id, s := range m.sessions {
		if s.lastAccess.Before(cutoff) {
			delete(m.sessions, id)
		}
	}
}

func (m *sessionManager) startJanitor(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.evictExpired()
			}
		}
	}()
}
