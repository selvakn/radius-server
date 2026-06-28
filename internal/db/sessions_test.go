package db_test

import (
	"testing"
	"time"
)

func TestSessionStartAndList(t *testing.T) {
	d := openTestDB(t)
	if err := d.UpsertSessionStart("s1", "alice", "10.0.0.1", "", time.Now()); err != nil {
		t.Fatalf("start: %v", err)
	}
	sessions, err := d.ListActiveSessions()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1, got %d", len(sessions))
	}
	if sessions[0].Username != "alice" || sessions[0].Status != "active" {
		t.Errorf("unexpected session: %+v", sessions[0])
	}
}

func TestSessionInterim(t *testing.T) {
	d := openTestDB(t)
	_ = d.UpsertSessionStart("s2", "bob", "10.0.0.1", "", time.Now())
	if err := d.UpdateSessionInterim("s2", 1000, 2000, 30); err != nil {
		t.Fatalf("interim: %v", err)
	}
	sessions, _ := d.ListActiveSessions()
	if sessions[0].BytesIn != 1000 || sessions[0].BytesOut != 2000 {
		t.Errorf("unexpected bytes: %+v", sessions[0])
	}
}

func TestSessionStop(t *testing.T) {
	d := openTestDB(t)
	_ = d.UpsertSessionStart("s3", "carol", "10.0.0.1", "", time.Now())
	if err := d.StopSession("s3", 500, 1500, 60, "User-Request", time.Now()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	active, _ := d.ListActiveSessions()
	if len(active) != 0 {
		t.Errorf("expected 0 active after stop")
	}
	byUser, _ := d.ListSessionsByUser("carol")
	if len(byUser) != 1 || byUser[0].Status != "stopped" {
		t.Errorf("expected stopped session for carol")
	}
}

func TestSessionUpsertIdempotent(t *testing.T) {
	d := openTestDB(t)
	_ = d.UpsertSessionStart("s4", "dave", "10.0.0.1", "", time.Now())
	_ = d.UpsertSessionStart("s4", "dave", "10.0.0.1", "", time.Now())
	sessions, _ := d.ListActiveSessions()
	if len(sessions) != 1 {
		t.Errorf("expected 1 session (upsert), got %d", len(sessions))
	}
}

func TestGetActiveSessionByID_Found(t *testing.T) {
	d := openTestDB(t)
	_ = d.UpsertSessionStart("s-active", "frank", "10.0.0.1", "", time.Now())
	sessions, _ := d.ListActiveSessions()
	if len(sessions) == 0 {
		t.Fatal("no active sessions")
	}
	got, err := d.GetActiveSessionByID(sessions[0].ID)
	if err != nil {
		t.Fatalf("expected session, got error: %v", err)
	}
	if got.Username != "frank" {
		t.Errorf("expected frank, got %q", got.Username)
	}
}

func TestGetActiveSessionByID_NotFound(t *testing.T) {
	d := openTestDB(t)
	_, err := d.GetActiveSessionByID(9999)
	if err == nil {
		t.Fatal("expected error for unknown id")
	}
}

func TestGetActiveSessionByID_InactiveReturnsNotFound(t *testing.T) {
	d := openTestDB(t)
	_ = d.UpsertSessionStart("s-stop", "grace", "10.0.0.1", "", time.Now())
	sessions, _ := d.ListActiveSessions()
	_ = d.StopSession("s-stop", 0, 0, 0, "User-Request", time.Now())
	_, err := d.GetActiveSessionByID(sessions[0].ID)
	if err == nil {
		t.Fatal("expected not found for stopped session")
	}
}

func TestGetActiveSessionsByUser_ReturnsOnlyActive(t *testing.T) {
	d := openTestDB(t)
	_ = d.UpsertSessionStart("sa1", "henry", "10.0.0.1", "", time.Now())
	_ = d.UpsertSessionStart("sa2", "henry", "10.0.0.1", "", time.Now())
	_ = d.UpsertSessionStart("sa3", "henry", "10.0.0.1", "", time.Now())
	_ = d.StopSession("sa3", 0, 0, 0, "User-Request", time.Now())

	active, err := d.GetActiveSessionsByUser("henry")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(active) != 2 {
		t.Errorf("expected 2 active sessions for henry, got %d", len(active))
	}
}

func TestListRecentSessions(t *testing.T) {
	d := openTestDB(t)
	for _, id := range []string{"r1", "r2", "r3"} {
		_ = d.UpsertSessionStart(id, "eve", "10.0.0.1", "", time.Now())
	}
	recent, err := d.ListRecentSessions(10)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(recent) != 3 {
		t.Errorf("expected 3, got %d", len(recent))
	}
}
