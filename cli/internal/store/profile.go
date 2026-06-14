package store

import (
	"database/sql"
	"strings"
)

type ProfileRepo struct {
	db *sql.DB
}

func (r *ProfileRepo) SetIdentity(name, email, preferred string) error {
	if preferred == "" {
		if fields := strings.Fields(name); len(fields) > 0 {
			preferred = fields[0]
		} else {
			preferred = "there"
		}
	}
	if err := r.Set("name", name); err != nil {
		return err
	}
	if err := r.Set("email", email); err != nil {
		return err
	}
	return r.Set("preferred_name", preferred)
}

func (r *ProfileRepo) Set(key, value string) error {
	_, err := r.db.Exec(
		`INSERT OR REPLACE INTO user_profile (key, value) VALUES (?, ?)`,
		key, value,
	)
	return err
}

func (r *ProfileRepo) Get(key string) (string, error) {
	var value string
	err := r.db.QueryRow(`SELECT value FROM user_profile WHERE key = ?`, key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (r *ProfileRepo) All() (map[string]string, error) {
	rows, err := r.db.Query(`SELECT key, value FROM user_profile`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profile := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		profile[k] = v
	}
	return profile, rows.Err()
}
