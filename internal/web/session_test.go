package web_test

import (
	"testing"
	"time"

	"github.com/selvakn/radius-server/internal/web"
)

func TestSessionCreate(t *testing.T) {
	store := web.NewSessionStore()
	token := store.Create("admin")
	if len(token) < 32 {
		t.Errorf("expected token len >= 32, got %d", len(token))
	}
}

func TestSessionGet_Valid(t *testing.T) {
	store := web.NewSessionStore()
	token := store.Create("alice")
	sess, ok := store.Get(token)
	if !ok {
		t.Fatal("expected session to exist")
	}
	if sess.AdminUsername != "alice" {
		t.Errorf("expected alice, got %q", sess.AdminUsername)
	}
}

func TestSessionGet_InvalidToken(t *testing.T) {
	store := web.NewSessionStore()
	_, ok := store.Get("notavalidtoken")
	if ok {
		t.Fatal("expected not found for invalid token")
	}
}

func TestSessionDelete(t *testing.T) {
	store := web.NewSessionStore()
	token := store.Create("bob")
	store.Delete(token)
	_, ok := store.Get(token)
	if ok {
		t.Fatal("expected session to be deleted")
	}
}

func TestSessionCreate_UniqueTokens(t *testing.T) {
	store := web.NewSessionStore()
	t1 := store.Create("u1")
	t2 := store.Create("u2")
	if t1 == t2 {
		t.Error("tokens should be unique")
	}
}

func TestSession_HasCSRFToken(t *testing.T) {
	store := web.NewSessionStore()
	token := store.Create("admin")
	sess, _ := store.Get(token)
	if sess.CSRFToken == "" {
		t.Error("expected non-empty CSRF token")
	}
}

func TestSession_ExpiresAt(t *testing.T) {
	store := web.NewSessionStore()
	token := store.Create("admin")
	sess, _ := store.Get(token)
	if !sess.ExpiresAt.After(time.Now()) {
		t.Error("expected future expiry")
	}
}
