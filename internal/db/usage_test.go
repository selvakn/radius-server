package db_test

import (
	"testing"
	"time"
)

func TestGetCurrentMonthUsage_WithData(t *testing.T) {
	d := openTestDB(t)
	// Create a session with data in the current month
	_ = d.UpsertSessionStart("u1", "alice", "10.0.0.1", "", time.Now())
	_ = d.UpdateSessionInterim("u1", 1000, 2000, 60)

	usage, err := d.GetCurrentMonthUsage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	u, ok := usage["alice"]
	if !ok {
		t.Fatal("expected usage entry for alice")
	}
	if u.BytesIn != 1000 {
		t.Errorf("expected BytesIn 1000, got %d", u.BytesIn)
	}
	if u.BytesOut != 2000 {
		t.Errorf("expected BytesOut 2000, got %d", u.BytesOut)
	}
}

func TestGetCurrentMonthUsage_NoData(t *testing.T) {
	d := openTestDB(t)
	usage, err := d.GetCurrentMonthUsage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(usage) != 0 {
		t.Errorf("expected empty map, got %d entries", len(usage))
	}
}

func TestGetCurrentMonthUsage_MultipleSessionsSameUser(t *testing.T) {
	d := openTestDB(t)
	_ = d.UpsertSessionStart("s1", "bob", "10.0.0.1", "", time.Now())
	_ = d.UpdateSessionInterim("s1", 500, 1000, 30)
	_ = d.UpsertSessionStart("s2", "bob", "10.0.0.1", "", time.Now())
	_ = d.UpdateSessionInterim("s2", 300, 700, 20)

	usage, err := d.GetCurrentMonthUsage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	u, ok := usage["bob"]
	if !ok {
		t.Fatal("expected usage entry for bob")
	}
	if u.BytesIn != 800 {
		t.Errorf("expected BytesIn 800, got %d", u.BytesIn)
	}
	if u.BytesOut != 1700 {
		t.Errorf("expected BytesOut 1700, got %d", u.BytesOut)
	}
}

func TestGetMonthlyUsageHistory_MultipleMonths(t *testing.T) {
	d := openTestDB(t)
	// Current month session
	_ = d.UpsertSessionStart("h1", "carol", "10.0.0.1", "", time.Now())
	_ = d.UpdateSessionInterim("h1", 100, 200, 60)
	// Session attributed to a past month via stopped_at
	pastMonth := time.Now().AddDate(0, -1, 0)
	_ = d.UpsertSessionStart("h2", "carol", "10.0.0.1", "", pastMonth)
	_ = d.StopSession("h2", 300, 600, 120, "User-Request", pastMonth)

	history, err := d.GetMonthlyUsageHistory("carol")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) < 1 {
		t.Fatalf("expected at least 1 history entry, got %d", len(history))
	}
	// Most recent month first
	if history[0].Month == "" {
		t.Error("expected non-empty month string")
	}
}

func TestGetMonthlyUsageHistory_Cap24(t *testing.T) {
	d := openTestDB(t)
	// Insert 30 sessions in different months
	for i := 0; i < 30; i++ {
		m := time.Now().AddDate(0, -i, 0)
		sid := "cap" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		_ = d.UpsertSessionStart(sid, "dave", "10.0.0.1", "", m)
		_ = d.StopSession(sid, 100, 200, 60, "User-Request", m)
	}
	history, err := d.GetMonthlyUsageHistory("dave")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) > 24 {
		t.Errorf("expected at most 24 entries, got %d", len(history))
	}
}

func TestGetMonthlyUsageHistory_NoData(t *testing.T) {
	d := openTestDB(t)
	history, err := d.GetMonthlyUsageHistory("nobody")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected empty history, got %d", len(history))
	}
}
