package store

import "database/sql"

var migrations = []string{
	`CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		fingerprint TEXT NOT NULL UNIQUE,
		source TEXT NOT NULL,
		member TEXT NOT NULL,
		category TEXT NOT NULL,
		title TEXT NOT NULL,
		detail TEXT,
		entry_date TEXT NOT NULL,
		priority TEXT NOT NULL DEFAULT 'info',
		link TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'open',
		first_seen_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL,
		resolved_at TEXT
	);
	CREATE TABLE IF NOT EXISTS source_health (
		source TEXT PRIMARY KEY,
		last_run TEXT,
		duration_ms INTEGER,
		status TEXT,
		error_message TEXT,
		entries_count INTEGER,
		run_id TEXT
	);
	CREATE TABLE IF NOT EXISTS runs (
		id TEXT PRIMARY KEY,
		started_at TEXT NOT NULL,
		finished_at TEXT,
		sources_total INTEGER,
		sources_ok INTEGER,
		sources_failed INTEGER
	);
	CREATE INDEX IF NOT EXISTS idx_items_source ON items(source);
	CREATE INDEX IF NOT EXISTS idx_items_status ON items(status);
	CREATE INDEX IF NOT EXISTS idx_items_member ON items(member);
	CREATE INDEX IF NOT EXISTS idx_items_first_seen ON items(first_seen_at);
	CREATE INDEX IF NOT EXISTS idx_items_resolved ON items(resolved_at);
	CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT NOT NULL,
		name TEXT NOT NULL,
		value REAL NOT NULL,
		recorded_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);
	CREATE INDEX IF NOT EXISTS idx_metrics_source_name ON metrics(source, name);
	CREATE INDEX IF NOT EXISTS idx_metrics_recorded_at ON metrics(recorded_at);`,

	`CREATE TABLE IF NOT EXISTS chat_sessions (
		id TEXT PRIMARY KEY,
		title TEXT,
		started_at TEXT NOT NULL,
		last_active_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS chat_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL REFERENCES chat_sessions(id),
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id);`,

	`CREATE TABLE IF NOT EXISTS agent_memory (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at TEXT NOT NULL,
		last_scrape_at TEXT,
		content TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_agent_memory_created ON agent_memory(created_at);`,

	`CREATE TABLE IF NOT EXISTS user_profile (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`,

	`CREATE TABLE IF NOT EXISTS acked_alerts (
		fingerprint TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		acked_at TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS token_usage (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source TEXT NOT NULL,
		model TEXT NOT NULL,
		prompt_tokens INTEGER NOT NULL DEFAULT 0,
		completion_tokens INTEGER NOT NULL DEFAULT 0,
		total_tokens INTEGER NOT NULL DEFAULT 0,
		recorded_at TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_token_usage_recorded ON token_usage(recorded_at);`,

	`CREATE TABLE IF NOT EXISTS team_members (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		fingerprint  TEXT NOT NULL UNIQUE,
		name         TEXT NOT NULL,
		email        TEXT NOT NULL DEFAULT '',
		aliases      TEXT NOT NULL DEFAULT '',
		role         TEXT NOT NULL DEFAULT '',
		department   TEXT NOT NULL DEFAULT '',
		branch       TEXT NOT NULL DEFAULT '',
		registration TEXT NOT NULL DEFAULT '',
		manager_id   INTEGER REFERENCES team_members(id),
		source       TEXT NOT NULL,
		first_seen_at TEXT NOT NULL,
		last_seen_at  TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_team_members_email        ON team_members(email);
	CREATE INDEX IF NOT EXISTS idx_team_members_registration ON team_members(registration);
	CREATE INDEX IF NOT EXISTS idx_team_members_manager_id   ON team_members(manager_id);`,

	`ALTER TABLE team_members ADD COLUMN manager_registration TEXT NOT NULL DEFAULT '';`,

	`UPDATE team_members SET source = 'manual' WHERE source = 'config';`,
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`)
	if err != nil {
		return err
	}

	var current int
	row := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`)
	if err := row.Scan(&current); err != nil {
		return err
	}

	for i := current; i < len(migrations); i++ {
		if _, err := db.Exec(migrations[i]); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO schema_version (version) VALUES (?)`, i+1); err != nil {
			return err
		}
	}

	return nil
}
