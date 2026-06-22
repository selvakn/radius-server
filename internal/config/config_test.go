package config_test

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/selvakn/radius-server/internal/config"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoad_Valid(t *testing.T) {
	path := writeConfig(t, `
radius:
  shared_secret: "mysecret"
  port: 1812
database:
  path: ./radius.db
web:
  port: 8080
  session_secret: "12345678901234567890123456789012"
admins:
  - username: admin
    password_hash: "$2a$12$placeholder"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Radius.SharedSecret != "mysecret" {
		t.Errorf("unexpected shared secret: %q", cfg.Radius.SharedSecret)
	}
	if cfg.Radius.Port != 1812 {
		t.Errorf("unexpected port: %d", cfg.Radius.Port)
	}
	if len(cfg.Admins) != 1 {
		t.Errorf("expected 1 admin, got %d", len(cfg.Admins))
	}
}

func TestLoad_MissingSharedSecret(t *testing.T) {
	path := writeConfig(t, `
radius:
  port: 1812
web:
  session_secret: "12345678901234567890123456789012"
admins:
  - username: admin
    password_hash: "$2a$12$placeholder"
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing shared_secret")
	}
}

func TestLoad_MissingSessionSecret(t *testing.T) {
	path := writeConfig(t, `
radius:
  shared_secret: "mysecret"
web:
  port: 8080
admins:
  - username: admin
    password_hash: "$2a$12$placeholder"
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing session_secret")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeConfig(t, `{invalid yaml:::`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_EmptyAdmins(t *testing.T) {
	path := writeConfig(t, `
radius:
  shared_secret: "mysecret"
web:
  session_secret: "12345678901234567890123456789012"
admins: []
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Admins) != 0 {
		t.Errorf("expected 0 admins")
	}
}

func TestLoad_DefaultPorts(t *testing.T) {
	path := writeConfig(t, `
radius:
  shared_secret: "mysecret"
web:
  session_secret: "12345678901234567890123456789012"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Radius.Port != 1812 {
		t.Errorf("expected default RADIUS port 1812, got %d", cfg.Radius.Port)
	}
	if cfg.Web.Port != 8080 {
		t.Errorf("expected default web port 8080, got %d", cfg.Web.Port)
	}
}

func TestAdminUser_CheckPassword_Valid(t *testing.T) {
	// bcrypt hash of "correctpass" with cost 4 (MinCost)
	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpass"), bcrypt.MinCost)
	a := config.AdminUser{Username: "admin", PasswordHash: string(hash)}
	if !a.CheckPassword("correctpass") {
		t.Error("expected CheckPassword to return true for correct password")
	}
}

func TestAdminUser_CheckPassword_Invalid(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpass"), bcrypt.MinCost)
	a := config.AdminUser{Username: "admin", PasswordHash: string(hash)}
	if a.CheckPassword("wrongpass") {
		t.Error("expected CheckPassword to return false for wrong password")
	}
}

func TestConfig_FindAdmin_Found(t *testing.T) {
	path := writeConfig(t, `
radius:
  shared_secret: "s"
web:
  session_secret: "12345678901234567890123456789012"
admins:
  - username: alice
    password_hash: "$2a$04$placeholder"
  - username: bob
    password_hash: "$2a$04$placeholder2"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	admin, ok := cfg.FindAdmin("alice")
	if !ok {
		t.Fatal("expected to find alice")
	}
	if admin.Username != "alice" {
		t.Errorf("expected alice, got %q", admin.Username)
	}
}

func TestConfig_FindAdmin_NotFound(t *testing.T) {
	path := writeConfig(t, `
radius:
  shared_secret: "s"
web:
  session_secret: "12345678901234567890123456789012"
admins:
  - username: admin
    password_hash: "$2a$12$placeholder"
`)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok := cfg.FindAdmin("unknown")
	if ok {
		t.Fatal("expected not found for unknown admin")
	}
}
