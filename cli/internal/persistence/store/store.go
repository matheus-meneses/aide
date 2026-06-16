package store

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB

	Items       *ItemRepo
	Metrics     *MetricRepo
	Runs        *RunRepo
	Chat        *ChatRepo
	Memory      *MemoryRepo
	Profile     *ProfileRepo
	Acks        *AckRepo
	Tokens      *TokenRepo
	Maintenance *MaintenanceRepo
	Team        *TeamRepo
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "aide.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &Store{
		db:          db,
		Items:       &ItemRepo{db: db},
		Metrics:     &MetricRepo{db: db},
		Runs:        &RunRepo{db: db},
		Chat:        &ChatRepo{db: db},
		Memory:      &MemoryRepo{db: db},
		Profile:     &ProfileRepo{db: db},
		Acks:        &AckRepo{db: db},
		Tokens:      &TokenRepo{db: db},
		Maintenance: &MaintenanceRepo{db: db},
		Team:        &TeamRepo{db: db},
	}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func Fingerprint(source, link, title, member string) string {
	var key string
	if link != "" {
		key = source + "|" + link
	} else {
		key = source + "|" + title + "|" + member
	}
	h := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", h[:16])
}
