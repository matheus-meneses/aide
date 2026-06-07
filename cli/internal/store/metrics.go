package store

import (
	"database/sql"
	"time"
)

type MetricRepo struct {
	db *sql.DB
}

func (r *MetricRepo) HistoricalOpenCounts(source string, days int) ([]DailyCount, error) {
	now := time.Now().UTC()
	var results []DailyCount

	for i := days - 1; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		dayEnd := time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, 0, time.UTC).Format(time.RFC3339)

		query := `SELECT COUNT(*) FROM items
			WHERE first_seen_at <= ? AND (resolved_at IS NULL OR resolved_at > ?)`
		args := []any{dayEnd, dayEnd}
		if source != "" {
			query += " AND source = ?"
			args = append(args, source)
		}

		var count int
		if err := r.db.QueryRow(query, args...).Scan(&count); err != nil {
			return nil, err
		}
		results = append(results, DailyCount{
			Date:  day.Format("2006-01-02"),
			Count: count,
		})
	}
	return results, nil
}

func (r *MetricRepo) AverageResolutionAge(source string, since time.Time) (float64, error) {
	query := `SELECT AVG(julianday(resolved_at) - julianday(first_seen_at))
		FROM items WHERE status = 'resolved' AND resolved_at >= ?`
	args := []any{since.Format(time.RFC3339)}
	if source != "" {
		query += " AND source = ?"
		args = append(args, source)
	}

	var avg sql.NullFloat64
	if err := r.db.QueryRow(query, args...).Scan(&avg); err != nil {
		return 0, err
	}
	if avg.Valid {
		return avg.Float64, nil
	}
	return 0, nil
}

func (r *MetricRepo) Record(source, name string, value float64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(
		`INSERT INTO metrics (source, name, value, recorded_at) VALUES (?, ?, ?, ?)`,
		source, name, value, now,
	)
	return err
}

func (r *MetricRepo) History(source, name string, days int) ([]DailyMetric, error) {
	since := time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
	rows, err := r.db.Query(`
		SELECT date(recorded_at) as day, AVG(value) as avg_value
		FROM metrics
		WHERE source = ? AND name = ? AND recorded_at >= ?
		GROUP BY day
		ORDER BY day
	`, source, name, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DailyMetric
	for rows.Next() {
		var dm DailyMetric
		if err := rows.Scan(&dm.Date, &dm.Value); err != nil {
			return nil, err
		}
		results = append(results, dm)
	}
	return results, rows.Err()
}

func (r *MetricRepo) Latest(source string) ([]Metric, error) {
	query := `
		SELECT m.id, m.source, m.name, m.value, m.recorded_at
		FROM metrics m
		INNER JOIN (
			SELECT source, name, MAX(recorded_at) as max_time
			FROM metrics
			GROUP BY source, name
		) latest ON m.source = latest.source AND m.name = latest.name AND m.recorded_at = latest.max_time
	`
	var args []any
	if source != "" {
		query += " WHERE m.source = ?"
		args = append(args, source)
	}
	query += " ORDER BY m.source, m.name"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Metric
	for rows.Next() {
		var m Metric
		if err := rows.Scan(&m.ID, &m.Source, &m.Name, &m.Value, &m.RecordedAt); err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, rows.Err()
}
