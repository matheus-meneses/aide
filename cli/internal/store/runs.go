package store

import "database/sql"

type RunRepo struct {
	db *sql.DB
}

func (r *RunRepo) UpsertHealth(h SourceHealth) error {
	_, err := r.db.Exec(`
		INSERT INTO source_health (source, last_run, duration_ms, status, error_message, entries_count, run_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source) DO UPDATE SET
			last_run = excluded.last_run,
			duration_ms = excluded.duration_ms,
			status = excluded.status,
			error_message = excluded.error_message,
			entries_count = excluded.entries_count,
			run_id = excluded.run_id
	`, h.Source, h.LastRun, h.DurationMs, h.Status, h.ErrorMessage, h.EntriesCount, h.RunID)
	return err
}

func (r *RunRepo) Insert(run Run) error {
	_, err := r.db.Exec(`
		INSERT INTO runs (id, started_at, finished_at, sources_total, sources_ok, sources_failed)
		VALUES (?, ?, ?, ?, ?, ?)
	`, run.ID, run.StartedAt, run.FinishedAt, run.SourcesTotal, run.SourcesOK, run.SourcesFailed)
	return err
}

func (r *RunRepo) Update(run Run) error {
	_, err := r.db.Exec(`
		UPDATE runs SET finished_at = ?, sources_total = ?, sources_ok = ?, sources_failed = ?
		WHERE id = ?
	`, run.FinishedAt, run.SourcesTotal, run.SourcesOK, run.SourcesFailed, run.ID)
	return err
}

func (r *RunRepo) AllHealth() ([]SourceHealth, error) {
	rows, err := r.db.Query(`SELECT source, last_run, duration_ms, status, error_message, entries_count, run_id FROM source_health ORDER BY source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SourceHealth
	for rows.Next() {
		var h SourceHealth
		if err := rows.Scan(&h.Source, &h.LastRun, &h.DurationMs, &h.Status, &h.ErrorMessage, &h.EntriesCount, &h.RunID); err != nil {
			return nil, err
		}
		results = append(results, h)
	}
	return results, rows.Err()
}

func (r *RunRepo) History(limit int) ([]Run, error) {
	rows, err := r.db.Query(`SELECT id, started_at, finished_at, sources_total, sources_ok, sources_failed FROM runs ORDER BY started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Run
	for rows.Next() {
		var run Run
		if err := rows.Scan(&run.ID, &run.StartedAt, &run.FinishedAt, &run.SourcesTotal, &run.SourcesOK, &run.SourcesFailed); err != nil {
			return nil, err
		}
		results = append(results, run)
	}
	return results, rows.Err()
}
