package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Session struct {
	ID             int64
	SessionID      string
	Username       string
	NasIP          string
	StartedAt      time.Time
	UpdatedAt      time.Time
	StoppedAt      *time.Time
	BytesIn        int64
	BytesOut       int64
	SessionTime    int64
	TerminateCause string
	Status         string
}

func (d *DB) UpsertSessionStart(sessionID, username, nasIP string, startedAt time.Time) error {
	_, err := d.sql.Exec(`
		INSERT INTO sessions (session_id, username, nas_ip, started_at, updated_at, status)
		VALUES (?, ?, ?, ?, ?, 'active')
		ON CONFLICT(session_id) DO UPDATE SET
			updated_at = excluded.updated_at,
			status     = 'active'`,
		sessionID, username, nasIP,
		startedAt.UTC().Format("2006-01-02 15:04:05"),
		startedAt.UTC().Format("2006-01-02 15:04:05"),
	)
	return err
}

func (d *DB) UpdateSessionInterim(sessionID string, bytesIn, bytesOut, sessionTime int64) error {
	_, err := d.sql.Exec(`
		UPDATE sessions
		SET bytes_in = ?, bytes_out = ?, session_time = ?, updated_at = ?
		WHERE session_id = ?`,
		bytesIn, bytesOut, sessionTime,
		time.Now().UTC().Format("2006-01-02 15:04:05"),
		sessionID,
	)
	return err
}

func (d *DB) StopSession(sessionID string, bytesIn, bytesOut, sessionTime int64, terminateCause string, stoppedAt time.Time) error {
	_, err := d.sql.Exec(`
		UPDATE sessions
		SET bytes_in = ?, bytes_out = ?, session_time = ?,
		    terminate_cause = ?, stopped_at = ?, updated_at = ?, status = 'stopped'
		WHERE session_id = ?`,
		bytesIn, bytesOut, sessionTime,
		terminateCause,
		stoppedAt.UTC().Format("2006-01-02 15:04:05"),
		stoppedAt.UTC().Format("2006-01-02 15:04:05"),
		sessionID,
	)
	return err
}

func (d *DB) ListActiveSessions() ([]Session, error) {
	rows, err := d.sql.Query(`
		SELECT id, session_id, username, nas_ip, started_at, updated_at,
		       stopped_at, bytes_in, bytes_out, session_time, terminate_cause, status
		FROM sessions WHERE status = 'active' ORDER BY started_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list active sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanSessions(rows)
}

func (d *DB) ListSessionsByUser(username string) ([]Session, error) {
	rows, err := d.sql.Query(`
		SELECT id, session_id, username, nas_ip, started_at, updated_at,
		       stopped_at, bytes_in, bytes_out, session_time, terminate_cause, status
		FROM sessions WHERE username = ? ORDER BY started_at DESC LIMIT 100`, username)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanSessions(rows)
}

func (d *DB) ListRecentSessions(limit int) ([]Session, error) {
	rows, err := d.sql.Query(`
		SELECT id, session_id, username, nas_ip, started_at, updated_at,
		       stopped_at, bytes_in, bytes_out, session_time, terminate_cause, status
		FROM sessions ORDER BY started_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanSessions(rows)
}

func scanSessions(rows *sql.Rows) ([]Session, error) {
	var sessions []Session
	for rows.Next() {
		var s Session
		var startedAt, updatedAt, stoppedAtRaw interface{}
		err := rows.Scan(&s.ID, &s.SessionID, &s.Username, &s.NasIP,
			&startedAt, &updatedAt, &stoppedAtRaw,
			&s.BytesIn, &s.BytesOut, &s.SessionTime,
			&s.TerminateCause, &s.Status)
		if err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		s.StartedAt = parseTime(startedAt)
		s.UpdatedAt = parseTime(updatedAt)
		if stoppedAtRaw != nil {
			t := parseTime(stoppedAtRaw)
			if !t.IsZero() {
				s.StoppedAt = &t
			}
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if sessions == nil {
		sessions = []Session{}
	}
	return sessions, nil
}

var ErrSessionNotFound = errors.New("session not found")

var timeFormats = []string{
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05",
	time.RFC3339,
	time.RFC3339Nano,
}

func parseTime(v interface{}) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		return parseTimeString(t)
	case []byte:
		return parseTimeString(string(t))
	}
	return time.Time{}
}

func parseTimeString(s string) time.Time {
	for _, f := range timeFormats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
