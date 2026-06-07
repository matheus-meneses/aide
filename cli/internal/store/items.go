package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type ItemRepo struct {
	db *sql.DB
}

func (r *ItemRepo) Upsert(source string, items []Item) (newCount, updatedCount int, err error) {
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := r.db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	upsertStmt, err := tx.Prepare(`
		INSERT INTO items (fingerprint, source, member, category, title, detail, entry_date, priority, link, status, first_seen_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'open', ?, ?)
		ON CONFLICT(fingerprint) DO UPDATE SET
			member = excluded.member,
			title = excluded.title,
			detail = excluded.detail,
			entry_date = excluded.entry_date,
			priority = excluded.priority,
			link = excluded.link,
			status = 'open',
			last_seen_at = excluded.last_seen_at,
			resolved_at = NULL
	`)
	if err != nil {
		return 0, 0, err
	}
	defer upsertStmt.Close()

	existing := make(map[string]bool)
	rows, err := tx.Query("SELECT fingerprint FROM items WHERE source = ?", source)
	if err != nil {
		return 0, 0, fmt.Errorf("loading existing fingerprints: %w", err)
	}
	for rows.Next() {
		var fp string
		if scanErr := rows.Scan(&fp); scanErr != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("scanning fingerprint: %w", scanErr)
		}
		existing[fp] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, 0, fmt.Errorf("iterating fingerprints: %w", err)
	}
	rows.Close()

	seenFingerprints := make(map[string]bool, len(items))

	for _, item := range items {
		fp := item.Fingerprint
		seenFingerprints[fp] = true

		_, execErr := upsertStmt.Exec(fp, source, item.Member, item.Category, item.Title, item.Detail, item.EntryDate, item.Priority, item.Link, now, now)
		if execErr != nil {
			return 0, 0, fmt.Errorf("upserting item: %w", execErr)
		}

		if existing[fp] {
			updatedCount++
		} else {
			newCount++
		}
	}

	if len(items) > 0 {
		if err := resolveUnseen(tx, source, now, seenFingerprints); err != nil {
			return 0, 0, fmt.Errorf("marking resolved: %w", err)
		}
	}

	return newCount, updatedCount, tx.Commit()
}

func resolveUnseen(tx *sql.Tx, source, now string, seen map[string]bool) error {
	const chunkSize = 400

	fps := make([]string, 0, len(seen))
	for fp := range seen {
		fps = append(fps, fp)
	}

	var clause strings.Builder
	args := []any{now, source}
	for start := 0; start < len(fps); start += chunkSize {
		end := start + chunkSize
		if end > len(fps) {
			end = len(fps)
		}
		clause.WriteString(" AND fingerprint NOT IN (")
		for i := start; i < end; i++ {
			if i > start {
				clause.WriteString(",")
			}
			clause.WriteString("?")
			args = append(args, fps[i])
		}
		clause.WriteString(")")
	}

	query := "UPDATE items SET status = 'resolved', resolved_at = ? WHERE source = ? AND status = 'open'" + clause.String()
	_, err := tx.Exec(query, args...)
	return err
}

func (r *ItemRepo) QueryOpen(source, member, category string) ([]Item, error) {
	query := `SELECT id, fingerprint, source, member, category, title, detail, entry_date, priority, link, status, first_seen_at, last_seen_at, COALESCE(resolved_at, '')
		FROM items WHERE status = 'open'`
	var args []any

	if source != "" {
		query += " AND source = ?"
		args = append(args, source)
	}
	if member != "" {
		query += " AND member = ?"
		args = append(args, member)
	}
	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}
	query += " ORDER BY source, entry_date DESC"

	return r.queryItems(query, args...)
}

func (r *ItemRepo) RecentlyResolved(source string, since time.Time) ([]Item, error) {
	query := `SELECT id, fingerprint, source, member, category, title, detail, entry_date, priority, link, status, first_seen_at, last_seen_at, COALESCE(resolved_at, '')
		FROM items WHERE status = 'resolved' AND resolved_at >= ?`
	args := []any{since.Format(time.RFC3339)}

	if source != "" {
		query += " AND source = ?"
		args = append(args, source)
	}
	query += " ORDER BY resolved_at DESC"

	return r.queryItems(query, args...)
}

func (r *ItemRepo) RecentlyDiscovered(source string, since time.Time) ([]Item, error) {
	query := `SELECT id, fingerprint, source, member, category, title, detail, entry_date, priority, link, status, first_seen_at, last_seen_at, COALESCE(resolved_at, '')
		FROM items WHERE first_seen_at >= ?`
	args := []any{since.Format(time.RFC3339)}

	if source != "" {
		query += " AND source = ?"
		args = append(args, source)
	}
	query += " ORDER BY first_seen_at DESC"

	return r.queryItems(query, args...)
}

func (r *ItemRepo) CountOpenBySource() (map[string]int, error) {
	rows, err := r.db.Query(`SELECT source, COUNT(*) FROM items WHERE status = 'open' GROUP BY source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var source string
		var count int
		if err := rows.Scan(&source, &count); err != nil {
			return nil, err
		}
		counts[source] = count
	}
	return counts, rows.Err()
}

func (r *ItemRepo) CountResolvedSince(since time.Time) (map[string]int, error) {
	rows, err := r.db.Query(`SELECT source, COUNT(*) FROM items WHERE status = 'resolved' AND resolved_at >= ? GROUP BY source`, since.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var source string
		var count int
		if err := rows.Scan(&source, &count); err != nil {
			return nil, err
		}
		counts[source] = count
	}
	return counts, rows.Err()
}

func (r *ItemRepo) Search(query string) ([]Item, error) {
	q := `SELECT id, fingerprint, source, member, category, title, detail, entry_date, priority, link, status, first_seen_at, last_seen_at, COALESCE(resolved_at, '')
		FROM items WHERE status = 'open' AND (title LIKE ? OR detail LIKE ? OR link LIKE ?) ORDER BY entry_date DESC LIMIT 20`
	pattern := "%" + query + "%"
	return r.queryItems(q, pattern, pattern, pattern)
}

func (r *ItemRepo) TodayEvents() ([]Item, error) {
	today := time.Now().Format("2006-01-02")
	q := `SELECT id, fingerprint, source, member, category, title, detail, entry_date, priority, link, status, first_seen_at, last_seen_at, COALESCE(resolved_at, '')
		FROM items WHERE status = 'open' AND category = 'event' AND entry_date = ? ORDER BY detail ASC`
	return r.queryItems(q, today)
}

func (r *ItemRepo) queryItems(query string, args ...any) ([]Item, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Fingerprint, &item.Source, &item.Member, &item.Category, &item.Title, &item.Detail, &item.EntryDate, &item.Priority, &item.Link, &item.Status, &item.FirstSeenAt, &item.LastSeenAt, &item.ResolvedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
