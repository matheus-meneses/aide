package store

import "database/sql"

type ProfileRepo struct {
	db *sql.DB
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
