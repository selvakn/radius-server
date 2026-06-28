package db

import "fmt"

type MonthlyUsage struct {
	Month    string
	BytesIn  int64
	BytesOut int64
}

func (d *DB) GetCurrentMonthUsage() (map[string]MonthlyUsage, error) {
	rows, err := d.sql.Query(`
		SELECT username,
		       SUM(bytes_in)  AS upload,
		       SUM(bytes_out) AS download
		FROM sessions
		WHERE strftime('%Y-%m', datetime(
		        CASE WHEN status = 'stopped' THEN stopped_at ELSE updated_at END,
		        'localtime')) = strftime('%Y-%m', 'now', 'localtime')
		GROUP BY username`)
	if err != nil {
		return nil, fmt.Errorf("current month usage: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]MonthlyUsage)
	for rows.Next() {
		var username string
		var u MonthlyUsage
		if err := rows.Scan(&username, &u.BytesIn, &u.BytesOut); err != nil {
			return nil, fmt.Errorf("scan usage: %w", err)
		}
		result[username] = u
	}
	return result, rows.Err()
}

func (d *DB) GetMonthlyUsageHistory(username string) ([]MonthlyUsage, error) {
	rows, err := d.sql.Query(`
		SELECT strftime('%Y-%m', datetime(
		         CASE WHEN status = 'stopped' THEN stopped_at ELSE updated_at END,
		         'localtime')) AS month,
		       SUM(bytes_in)  AS upload,
		       SUM(bytes_out) AS download
		FROM sessions
		WHERE username = ?
		GROUP BY month
		ORDER BY month DESC
		LIMIT 24`, username)
	if err != nil {
		return nil, fmt.Errorf("monthly usage history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var history []MonthlyUsage
	for rows.Next() {
		var u MonthlyUsage
		if err := rows.Scan(&u.Month, &u.BytesIn, &u.BytesOut); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		history = append(history, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if history == nil {
		history = []MonthlyUsage{}
	}
	return history, nil
}
