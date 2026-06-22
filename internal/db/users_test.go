package db_test

import (
	"path/filepath"
	"testing"

	"github.com/selvakn/radius-server/internal/db"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	d, err := db.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestCreateUser_Success(t *testing.T) {
	d := openTestDB(t)
	u := db.User{
		Username:     "alice",
		PasswordHash: "$2a$12$placeholder",
		Enabled:      true,
	}
	if err := d.CreateUser(u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	d := openTestDB(t)
	u := db.User{Username: "bob", PasswordHash: "hash", Enabled: true}
	if err := d.CreateUser(u); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := d.CreateUser(u); err == nil {
		t.Fatal("expected error for duplicate username")
	}
}

func TestGetUserByUsername_Found(t *testing.T) {
	d := openTestDB(t)
	u := db.User{Username: "carol", PasswordHash: "hash", Enabled: true}
	if err := d.CreateUser(u); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := d.GetUserByUsername("carol")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Username != "carol" {
		t.Errorf("expected carol, got %q", got.Username)
	}
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	d := openTestDB(t)
	_, err := d.GetUserByUsername("nobody")
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestListUsers(t *testing.T) {
	d := openTestDB(t)
	for _, name := range []string{"u1", "u2", "u3"} {
		if err := d.CreateUser(db.User{Username: name, PasswordHash: "h", Enabled: true}); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}
	users, err := d.ListUsers()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("expected 3 users, got %d", len(users))
	}
}

func TestSetEnabled(t *testing.T) {
	d := openTestDB(t)
	if err := d.CreateUser(db.User{Username: "dave", PasswordHash: "h", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	u, _ := d.GetUserByUsername("dave")
	if err := d.SetEnabled(u.ID, false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	u2, _ := d.GetUserByUsername("dave")
	if u2.Enabled {
		t.Error("expected user to be disabled")
	}
	if err := d.SetEnabled(u.ID, true); err != nil {
		t.Fatalf("enable: %v", err)
	}
	u3, _ := d.GetUserByUsername("dave")
	if !u3.Enabled {
		t.Error("expected user to be enabled")
	}
}

func TestUpdateUser(t *testing.T) {
	d := openTestDB(t)
	if err := d.CreateUser(db.User{Username: "eve", PasswordHash: "hash1", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	u, _ := d.GetUserByUsername("eve")
	down, up := 1024, 512
	upd := db.UserUpdate{PasswordHash: "hash2", DownloadRate: &down, UploadRate: &up}
	if err := d.UpdateUser(u.ID, upd); err != nil {
		t.Fatalf("update: %v", err)
	}
	u2, _ := d.GetUserByUsername("eve")
	if u2.PasswordHash != "hash2" {
		t.Errorf("expected updated hash")
	}
	if u2.DownloadRate == nil || *u2.DownloadRate != 1024 {
		t.Errorf("expected download rate 1024")
	}
}

func TestDeleteUser(t *testing.T) {
	d := openTestDB(t)
	if err := d.CreateUser(db.User{Username: "frank", PasswordHash: "h", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	u, _ := d.GetUserByUsername("frank")
	if err := d.DeleteUser(u.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := d.GetUserByUsername("frank")
	if err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestNullRates(t *testing.T) {
	d := openTestDB(t)
	if err := d.CreateUser(db.User{Username: "grace", PasswordHash: "h", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	u, _ := d.GetUserByUsername("grace")
	if u.DownloadRate != nil || u.UploadRate != nil {
		t.Error("expected nil rates for user without bandwidth limits")
	}
}
