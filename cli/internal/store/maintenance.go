package store

import (
	"database/sql"
	"fmt"
	"time"
)

type MaintenanceRepo struct {
	db *sql.DB
}

func (r *MaintenanceRepo) PruneCounts(keepDays int) (*PruneResult, error) {
	cutoff := time.Now().AddDate(0, 0, -keepDays).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")
	result := &PruneResult{}

	count := func(query string, args ...any) (int64, error) {
		var n int64
		if err := r.db.QueryRow(query, args...).Scan(&n); err != nil {
			return 0, err
		}
		return n, nil
	}

	n, err := count(`SELECT COUNT(*) FROM items WHERE status = 'resolved' AND resolved_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("counting resolved items: %w", err)
	}
	result.Items = n

	n, err = count(`SELECT COUNT(*) FROM items WHERE status = 'open' AND entry_date < ? AND entry_date < ? AND category != 'event'`, cutoff, today)
	if err != nil {
		return nil, fmt.Errorf("counting stale open items: %w", err)
	}
	result.Items += n

	result.Messages, err = count(`SELECT COUNT(*) FROM chat_messages WHERE created_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("counting messages: %w", err)
	}

	result.Sessions, err = count(`SELECT COUNT(*) FROM chat_sessions WHERE last_active_at < ? AND id NOT IN (SELECT DISTINCT session_id FROM chat_messages)`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("counting sessions: %w", err)
	}

	result.Memories, err = count(`SELECT COUNT(*) FROM agent_memory WHERE id NOT IN (SELECT id FROM agent_memory ORDER BY created_at DESC LIMIT 5) AND created_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("counting memories: %w", err)
	}

	result.Metrics, err = count(`SELECT COUNT(*) FROM metrics WHERE recorded_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("counting metrics: %w", err)
	}

	result.Runs, err = count(`SELECT COUNT(*) FROM runs WHERE started_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("counting runs: %w", err)
	}

	result.Acks, err = count(`SELECT COUNT(*) FROM acked_alerts WHERE acked_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("counting acks: %w", err)
	}

	result.Tokens, err = count(`SELECT COUNT(*) FROM token_usage WHERE recorded_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("counting tokens: %w", err)
	}

	return result, nil
}

func (r *MaintenanceRepo) Prune(keepDays int) (*PruneResult, error) {
	cutoff := time.Now().AddDate(0, 0, -keepDays).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")
	result := &PruneResult{}

	execCount := func(query string, args ...any) (int64, error) {
		res, err := r.db.Exec(query, args...)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		return n, nil
	}

	n, err := execCount(`DELETE FROM items WHERE status = 'resolved' AND resolved_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("pruning resolved items: %w", err)
	}
	result.Items = n

	n, err = execCount(`DELETE FROM items WHERE status = 'open' AND entry_date < ? AND entry_date < ? AND category != 'event'`, cutoff, today)
	if err != nil {
		return nil, fmt.Errorf("pruning stale open items: %w", err)
	}
	result.Items += n

	result.Messages, err = execCount(`DELETE FROM chat_messages WHERE created_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("pruning messages: %w", err)
	}

	result.Sessions, err = execCount(`DELETE FROM chat_sessions WHERE last_active_at < ? AND id NOT IN (SELECT DISTINCT session_id FROM chat_messages)`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("pruning sessions: %w", err)
	}

	result.Memories, err = execCount(`DELETE FROM agent_memory WHERE id NOT IN (SELECT id FROM agent_memory ORDER BY created_at DESC LIMIT 5) AND created_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("pruning memories: %w", err)
	}

	result.Metrics, err = execCount(`DELETE FROM metrics WHERE recorded_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("pruning metrics: %w", err)
	}

	result.Runs, err = execCount(`DELETE FROM runs WHERE started_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("pruning runs: %w", err)
	}

	result.Acks, err = execCount(`DELETE FROM acked_alerts WHERE acked_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("pruning acks: %w", err)
	}

	result.Tokens, err = execCount(`DELETE FROM token_usage WHERE recorded_at < ?`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("pruning tokens: %w", err)
	}

	return result, nil
}
