package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("not found")

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Enabled      bool
	DownloadRate *int
	UploadRate   *int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserUpdate struct {
	PasswordHash string
	DownloadRate *int
	UploadRate   *int
}

func (d *DB) CreateUser(u User) error {
	_, err := d.sql.Exec(
		`INSERT INTO users (username, password_hash, enabled, download_rate, upload_rate) VALUES (?, ?, ?, ?, ?)`,
		u.Username, u.PasswordHash, boolToInt(u.Enabled), u.DownloadRate, u.UploadRate,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (d *DB) GetUserByUsername(username string) (*User, error) {
	row := d.sql.QueryRow(
		`SELECT id, username, password_hash, enabled, download_rate, upload_rate, created_at, updated_at FROM users WHERE username = ?`,
		username,
	)
	return scanUser(row)
}

func (d *DB) GetUserByID(id int64) (*User, error) {
	row := d.sql.QueryRow(
		`SELECT id, username, password_hash, enabled, download_rate, upload_rate, created_at, updated_at FROM users WHERE id = ?`,
		id,
	)
	return scanUser(row)
}

func (d *DB) ListUsers() ([]User, error) {
	rows, err := d.sql.Query(
		`SELECT id, username, password_hash, enabled, download_rate, upload_rate, created_at, updated_at FROM users ORDER BY username`,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var users []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

func (d *DB) UpdateUser(id int64, upd UserUpdate) error {
	_, err := d.sql.Exec(
		`UPDATE users SET password_hash = ?, download_rate = ?, upload_rate = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		upd.PasswordHash, upd.DownloadRate, upd.UploadRate, id,
	)
	return err
}

func (d *DB) SetEnabled(id int64, enabled bool) error {
	_, err := d.sql.Exec(
		`UPDATE users SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		boolToInt(enabled), id,
	)
	return err
}

func (d *DB) DeleteUser(id int64) error {
	_, err := d.sql.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(s scanner) (*User, error) {
	var u User
	var enabled int
	var createdAt, updatedAt interface{}
	err := s.Scan(&u.ID, &u.Username, &u.PasswordHash, &enabled, &u.DownloadRate, &u.UploadRate, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	u.Enabled = enabled != 0
	u.CreatedAt = parseTime(createdAt)
	u.UpdatedAt = parseTime(updatedAt)
	return &u, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
