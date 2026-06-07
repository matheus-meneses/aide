package store

import (
	"database/sql"
	"time"
)

type MemoryRepo struct {
	db *sql.DB
}

func (r *MemoryRepo) Save(lastScrapeAt, content string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(
		`INSERT INTO agent_memory (created_at, last_scrape_at, content) VALUES (?, ?, ?)`,
		now, lastScrapeAt, content,
	)
	return err
}

func (r *MemoryRepo) LoadLast() (*AgentMemory, error) {
	var m AgentMemory
	err := r.db.QueryRow(
		`SELECT id, created_at, COALESCE(last_scrape_at, ''), content FROM agent_memory ORDER BY created_at DESC LIMIT 1`,
	).Scan(&m.ID, &m.CreatedAt, &m.LastScrapeAt, &m.Content)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MemoryRepo) Prune(keep int) error {
	_, err := r.db.Exec(
		`DELETE FROM agent_memory WHERE id NOT IN (SELECT id FROM agent_memory ORDER BY created_at DESC LIMIT ?)`,
		keep,
	)
	return err
}
