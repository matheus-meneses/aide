package store

import (
	"database/sql"
	"time"
)

type AckRepo struct {
	db *sql.DB
}

func (r *AckRepo) Add(fingerprint, title string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(
		`INSERT OR REPLACE INTO acked_alerts (fingerprint, title, acked_at) VALUES (?, ?, ?)`,
		fingerprint, title, now,
	)
	return err
}

func (r *AckRepo) ListActive() ([]AckedAlert, error) {
	cutoff := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	rows, err := r.db.Query(`SELECT fingerprint, title, acked_at FROM acked_alerts WHERE acked_at > ?`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var acks []AckedAlert
	for rows.Next() {
		var a AckedAlert
		if err := rows.Scan(&a.Fingerprint, &a.Title, &a.AckedAt); err != nil {
			return nil, err
		}
		acks = append(acks, a)
	}
	return acks, rows.Err()
}

func (r *AckRepo) Prune() error {
	cutoff := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	_, err := r.db.Exec(`DELETE FROM acked_alerts WHERE acked_at < ?`, cutoff)
	return err
}
