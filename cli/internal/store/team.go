package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type TeamRepo struct {
	db *sql.DB
}

func MemberFingerprint(name, registration, email string) string {
	var key string
	switch {
	case registration != "":
		key = name + "|" + registration
	case email != "":
		key = name + "|" + email
	default:
		key = name
	}
	h := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", h[:16])
}

func (r *TeamRepo) Upsert(members []Member) error {
	if len(members) == 0 {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // rollback on defer is a no-op after Commit and safe to ignore

	for i := range members {
		if members[i].Fingerprint == "" {
			members[i].Fingerprint = MemberFingerprint(members[i].Name, members[i].Registration, members[i].Email)
		}
		if members[i].FirstSeenAt == "" {
			members[i].FirstSeenAt = now
		}
		members[i].LastSeenAt = now

		_, err := tx.Exec(
			`
			INSERT INTO team_members (fingerprint, name, email, aliases, role, department, branch, registration, manager_id, manager_registration, source, first_seen_at, last_seen_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?, ?, ?)
			ON CONFLICT(fingerprint) DO UPDATE SET
				name                 = excluded.name,
				email                = excluded.email,
				aliases              = excluded.aliases,
				role                 = excluded.role,
				department           = excluded.department,
				branch               = excluded.branch,
				registration         = excluded.registration,
				manager_registration = excluded.manager_registration,
				source               = excluded.source,
				last_seen_at         = excluded.last_seen_at`,
			members[i].Fingerprint,
			members[i].Name,
			members[i].Email,
			members[i].Aliases,
			members[i].Role,
			members[i].Department,
			members[i].Branch,
			members[i].Registration,
			members[i].ManagerRef,
			members[i].Source,
			members[i].FirstSeenAt,
			now,
		)
		if err != nil {
			return fmt.Errorf("upsert member %q: %w", members[i].Name, err)
		}
	}

	for _, m := range members {
		if m.ManagerRef == "" {
			continue
		}
		var managerID int64
		err := tx.QueryRow(
			`SELECT id FROM team_members WHERE registration = ? AND registration != ''`,
			m.ManagerRef,
		).Scan(&managerID)
		if err == sql.ErrNoRows {
			err = tx.QueryRow(
				`SELECT id FROM team_members WHERE name = ?`,
				m.ManagerRef,
			).Scan(&managerID)
		}
		if err != nil {
			continue
		}
		if _, err := tx.Exec(
			`UPDATE team_members SET manager_id = ? WHERE fingerprint = ?`,
			managerID, m.Fingerprint,
		); err != nil {
			return fmt.Errorf("setting manager for %q: %w", m.Name, err)
		}
	}

	source := members[0].Source
	if source != "config" {
		fps := make([]string, len(members))
		for i, m := range members {
			fps[i] = m.Fingerprint
		}
		placeholders := strings.Repeat("?,", len(fps))
		placeholders = placeholders[:len(placeholders)-1]
		args := make([]any, 0, 2+len(fps))
		args = append(args, now, source)
		for _, fp := range fps {
			args = append(args, fp)
		}
		if _, err := tx.Exec(
			`UPDATE team_members SET last_seen_at = ?, manager_id = NULL
			 WHERE source = ? AND fingerprint NOT IN (`+placeholders+`)`,
			args...,
		); err != nil {
			return fmt.Errorf("orphaning removed members: %w", err)
		}
	}

	return tx.Commit()
}

func (r *TeamRepo) All() ([]Member, error) {
	rows, err := r.db.Query(`
		SELECT id, fingerprint, name, email, aliases, role, department, branch, registration,
		       manager_id, manager_registration, source, first_seen_at, last_seen_at
		FROM team_members
		ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(
			&m.ID, &m.Fingerprint, &m.Name, &m.Email, &m.Aliases,
			&m.Role, &m.Department, &m.Branch, &m.Registration,
			&m.ManagerID, &m.ManagerRegistration, &m.Source, &m.FirstSeenAt, &m.LastSeenAt,
		); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *TeamRepo) ReresolveManagers() (int, error) {
	rows, err := r.db.Query(`
		SELECT id, fingerprint, manager_registration
		FROM team_members
		WHERE manager_registration != ''`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type pending struct {
		id          int64
		fingerprint string
		managerReg  string
	}
	var batch []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.fingerprint, &p.managerReg); err != nil {
			return 0, err
		}
		batch = append(batch, p)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck // rollback on defer is a no-op after Commit and safe to ignore

	var updated int
	for _, p := range batch {
		var managerID int64
		err := tx.QueryRow(
			`SELECT id FROM team_members WHERE registration = ? AND registration != ''`,
			p.managerReg,
		).Scan(&managerID)
		if err == sql.ErrNoRows {
			err = tx.QueryRow(
				`SELECT id FROM team_members WHERE name = ?`,
				p.managerReg,
			).Scan(&managerID)
		}
		if err != nil {
			continue
		}
		if _, err := tx.Exec(
			`UPDATE team_members SET manager_id = ? WHERE fingerprint = ?`,
			managerID, p.fingerprint,
		); err != nil {
			return updated, err
		}
		updated++
	}

	return updated, tx.Commit()
}

func (r *TeamRepo) Resolve(alias string) string {
	if alias == "" {
		return alias
	}

	var name string
	err := r.db.QueryRow(
		`SELECT name FROM team_members WHERE name = ? OR email = ? OR registration = ?`,
		alias, alias, alias,
	).Scan(&name)
	if err == nil {
		return name
	}

	rows, err := r.db.Query(`SELECT name, aliases FROM team_members WHERE aliases != ''`)
	if err != nil {
		return alias
	}
	defer rows.Close()

	for rows.Next() {
		var memberName, aliasesJSON string
		if err := rows.Scan(&memberName, &aliasesJSON); err != nil {
			continue
		}
		var aliases []string
		if err := json.Unmarshal([]byte(aliasesJSON), &aliases); err != nil {
			continue
		}
		for _, a := range aliases {
			if strings.EqualFold(a, alias) {
				return memberName
			}
		}
	}

	return alias
}
