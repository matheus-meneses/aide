package store

import (
	"database/sql"
	"fmt"
	"time"
)

type ChatRepo struct {
	db *sql.DB
}

func (r *ChatRepo) CreateSession(id, startedAt string) error {
	_, err := r.db.Exec(
		`INSERT INTO chat_sessions (id, title, started_at, last_active_at) VALUES (?, '', ?, ?)
		 ON CONFLICT(id) DO NOTHING`,
		id, startedAt, startedAt,
	)
	return err
}

func (r *ChatRepo) UpdateSessionTitle(id, title string) error {
	_, err := r.db.Exec(`UPDATE chat_sessions SET title = ? WHERE id = ?`, title, id)
	return err
}

func (r *ChatRepo) TouchSession(id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(`UPDATE chat_sessions SET last_active_at = ? WHERE id = ?`, now, id)
	return err
}

func (r *ChatRepo) InsertMessage(sessionID, role, content, createdAt string) error {
	_, err := r.db.Exec(
		`INSERT INTO chat_messages (session_id, role, content, created_at) VALUES (?, ?, ?, ?)`,
		sessionID, role, content, createdAt,
	)
	if err == nil {
		if touchErr := r.TouchSession(sessionID); touchErr != nil {
			return fmt.Errorf("touching chat session: %w", touchErr)
		}
	}
	return err
}

func (r *ChatRepo) ListSessions(limit int) ([]ChatSession, error) {
	rows, err := r.db.Query(
		`SELECT id, COALESCE(title, ''), started_at, last_active_at FROM chat_sessions ORDER BY last_active_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []ChatSession
	for rows.Next() {
		var sess ChatSession
		if err := rows.Scan(&sess.ID, &sess.Title, &sess.StartedAt, &sess.LastActiveAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

func (r *ChatRepo) LoadMessages(sessionID string) ([]ChatMessage, error) {
	rows, err := r.db.Query(
		`SELECT id, session_id, role, content, created_at FROM chat_messages WHERE session_id = ? ORDER BY id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
