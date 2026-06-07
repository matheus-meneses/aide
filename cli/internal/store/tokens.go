package store

import (
	"database/sql"
	"fmt"
	"time"
)

type TokenRepo struct {
	db *sql.DB
}

func (r *TokenRepo) Record(source, model string, prompt, completion, total int) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(
		`INSERT INTO token_usage (source, model, prompt_tokens, completion_tokens, total_tokens, recorded_at) VALUES (?, ?, ?, ?, ?, ?)`,
		source, model, prompt, completion, total, now,
	)
	return err
}

func (r *TokenRepo) Stats() (*TokenSummary, error) {
	today := time.Now().Format("2006-01-02")
	weekAgo := time.Now().Add(-7 * 24 * time.Hour).UTC().Format(time.RFC3339)

	summary := &TokenSummary{BySource: make(map[string]int)}

	if err := r.db.QueryRow(`SELECT COALESCE(SUM(total_tokens), 0) FROM token_usage WHERE recorded_at >= ?`, today).Scan(&summary.TodayTokens); err != nil {
		return nil, fmt.Errorf("querying today tokens: %w", err)
	}
	if err := r.db.QueryRow(`SELECT COALESCE(SUM(total_tokens), 0) FROM token_usage WHERE recorded_at >= ?`, weekAgo).Scan(&summary.WeekTokens); err != nil {
		return nil, fmt.Errorf("querying week tokens: %w", err)
	}
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM token_usage WHERE recorded_at >= ?`, weekAgo).Scan(&summary.TotalCalls); err != nil {
		return nil, fmt.Errorf("querying call count: %w", err)
	}

	if summary.TotalCalls > 0 {
		summary.AvgPerDay = summary.WeekTokens / 7
	}

	rows, err := r.db.Query(`SELECT source, COALESCE(SUM(total_tokens), 0) FROM token_usage WHERE recorded_at >= ? GROUP BY source`, weekAgo)
	if err != nil {
		return nil, fmt.Errorf("querying by source: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var src string
		var total int
		if err := rows.Scan(&src, &total); err != nil {
			return nil, fmt.Errorf("scanning token row: %w", err)
		}
		summary.BySource[src] = total
	}

	return summary, rows.Err()
}

func (r *TokenRepo) DailyStats(days int) ([]DailyTokens, error) {
	var result []DailyTokens
	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		nextDate := time.Now().AddDate(0, 0, -i+1).Format("2006-01-02")

		var agent, chat int
		if err := r.db.QueryRow(`SELECT COALESCE(SUM(total_tokens), 0) FROM token_usage WHERE source = 'agent' AND recorded_at >= ? AND recorded_at < ?`, date, nextDate).Scan(&agent); err != nil {
			return nil, fmt.Errorf("daily agent tokens: %w", err)
		}
		if err := r.db.QueryRow(`SELECT COALESCE(SUM(total_tokens), 0) FROM token_usage WHERE source = 'chat' AND recorded_at >= ? AND recorded_at < ?`, date, nextDate).Scan(&chat); err != nil {
			return nil, fmt.Errorf("daily chat tokens: %w", err)
		}

		result = append(result, DailyTokens{Date: date, Agent: agent, Chat: chat})
	}
	return result, nil
}
