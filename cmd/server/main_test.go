package main

import (
	"strings"
	"testing"
)

func TestHashPassword_ValidInput(t *testing.T) {
	in := strings.NewReader("mypassword\n")
	var out strings.Builder
	if err := hashPassword(in, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := strings.TrimSpace(out.String())
	if !strings.HasPrefix(result, "$2a$") {
		t.Errorf("expected bcrypt hash, got: %q", result)
	}
}

func TestHashPassword_EmptyInput(t *testing.T) {
	in := strings.NewReader("\n")
	var out strings.Builder
	if err := hashPassword(in, &out); err == nil {
		t.Fatal("expected error for empty password")
	}
}
