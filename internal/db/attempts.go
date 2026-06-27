package db

import (
	"fmt"
	"time"
)

type AttemptSummary struct {
	Username        string
	Count24h        int
	LastAttemptedAt time.Time
	LastOutcome     string
	LastPassword    string
	IsKnown         bool
	UserID          int64
}

func (d *DB) RecordAttempt(username, outcome, password string) error {
	return d.RecordAttemptAt(username, outcome, password, time.Now())
}

func (d *DB) RecordAttemptAt(username, outcome, password string, at time.Time) error {
	_, err := d.sql.Exec(
		`INSERT INTO auth_attempts (username, attempted_at, outcome, attempted_password) VALUES (?, ?, ?, ?)`,
		username, at.UTC().Format("2006-01-02 15:04:05"), outcome, password,
	)
	if err != nil {
		return fmt.Errorf("record attempt: %w", err)
	}
	return nil
}

func (d *DB) ListAttemptSummaries() ([]AttemptSummary, error) {
	rows, err := d.sql.Query(`
		SELECT
			a.username,
			COUNT(CASE WHEN a.attempted_at >= datetime('now', '-24 hours') THEN 1 END) AS count_24h,
			MAX(a.attempted_at) AS last_attempted_at,
			(SELECT outcome           FROM auth_attempts WHERE username = a.username ORDER BY attempted_at DESC, id DESC LIMIT 1) AS last_outcome,
			(SELECT attempted_password FROM auth_attempts WHERE username = a.username AND outcome = 'rejected' ORDER BY attempted_at DESC, id DESC LIMIT 1) AS last_password,
			CASE WHEN u.username IS NOT NULL THEN 1 ELSE 0 END AS is_known,
			COALESCE(u.id, 0) AS user_id
		FROM auth_attempts a
		LEFT JOIN users u ON u.username = a.username
		GROUP BY a.username
		ORDER BY MAX(a.attempted_at) DESC
		LIMIT 200`)
	if err != nil {
		return nil, fmt.Errorf("list attempt summaries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var summaries []AttemptSummary
	for rows.Next() {
		var s AttemptSummary
		var lastAt interface{}
		var isKnown int
		var lastPassword *string
		if err := rows.Scan(&s.Username, &s.Count24h, &lastAt, &s.LastOutcome, &lastPassword, &isKnown, &s.UserID); err != nil {
			return nil, fmt.Errorf("scan attempt summary: %w", err)
		}
		s.LastAttemptedAt = parseTime(lastAt)
		s.IsKnown = isKnown != 0
		if lastPassword != nil {
			s.LastPassword = *lastPassword
		}
		summaries = append(summaries, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if summaries == nil {
		summaries = []AttemptSummary{}
	}
	return summaries, nil
}

func (d *DB) PurgeOldAttempts() error {
	_, err := d.sql.Exec(
		`DELETE FROM auth_attempts WHERE attempted_at < datetime('now', '-7 days')`,
	)
	if err != nil {
		return fmt.Errorf("purge old attempts: %w", err)
	}
	return nil
}
