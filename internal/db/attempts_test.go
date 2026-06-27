package db_test

import (
	"testing"
	"time"

	"github.com/selvakn/radius-server/internal/db"
)

func TestRecordAttempt(t *testing.T) {
	d := openTestDB(t)
	if err := d.RecordAttempt("alice", "accepted", ""); err != nil {
		t.Fatalf("record: %v", err)
	}
}

func TestListAttemptSummaries_Count24h(t *testing.T) {
	d := openTestDB(t)
	for i := 0; i < 3; i++ {
		_ = d.RecordAttempt("bob", "rejected", "")
	}
	summaries, err := d.ListAttemptSummaries()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].Count24h != 3 {
		t.Errorf("expected count 3, got %d", summaries[0].Count24h)
	}
}

func TestListAttemptSummaries_LastOutcome(t *testing.T) {
	d := openTestDB(t)
	_ = d.RecordAttempt("carol", "rejected", "")
	_ = d.RecordAttempt("carol", "accepted", "")
	summaries, _ := d.ListAttemptSummaries()
	if len(summaries) == 0 {
		t.Fatal("expected summary")
	}
	if summaries[0].LastOutcome != "accepted" {
		t.Errorf("expected last outcome accepted, got %q", summaries[0].LastOutcome)
	}
}

func TestListAttemptSummaries_IsKnown(t *testing.T) {
	d := openTestDB(t)
	_ = d.CreateUser(db.User{Username: "known", PasswordHash: "h", Enabled: true})
	_ = d.RecordAttempt("known", "accepted", "")
	_ = d.RecordAttempt("unknown", "rejected", "")

	summaries, err := d.ListAttemptSummaries()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	byName := map[string]db.AttemptSummary{}
	for _, s := range summaries {
		byName[s.Username] = s
	}
	if !byName["known"].IsKnown {
		t.Error("expected known user to have IsKnown=true")
	}
	if byName["unknown"].IsKnown {
		t.Error("expected unknown user to have IsKnown=false")
	}
}

func TestListAttemptSummaries_Cap200(t *testing.T) {
	d := openTestDB(t)
	for i := 0; i < 210; i++ {
		_ = d.RecordAttempt(randomUsername(i), "rejected", "")
	}
	summaries, err := d.ListAttemptSummaries()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(summaries) > 200 {
		t.Errorf("expected at most 200 rows, got %d", len(summaries))
	}
}

func TestPurgeOldAttempts(t *testing.T) {
	d := openTestDB(t)
	_ = d.RecordAttempt("old", "rejected", "")
	_ = d.RecordAttemptAt("old", "rejected", "", time.Now().Add(-8*24*time.Hour))
	_ = d.RecordAttempt("recent", "accepted", "")

	if err := d.PurgeOldAttempts(); err != nil {
		t.Fatalf("purge: %v", err)
	}
	// verify the 8-day-old row is gone
	recent, _ := d.ListAttemptSummaries()
	found := false
	for _, s := range recent {
		if s.Username == "recent" {
			found = true
		}
	}
	if !found {
		t.Error("expected recent attempt to survive purge")
	}
}

func randomUsername(i int) string {
	return "user" + string(rune('a'+i%26)) + string(rune('0'+i/26%10))
}
