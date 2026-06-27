package accounting_test

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"

	"github.com/selvakn/radius-server/internal/accounting"
	"github.com/selvakn/radius-server/internal/db"
)

const testSecret = "testsecret"

func openDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func startServer(t *testing.T, d *db.DB) string {
	t.Helper()
	h := accounting.New(d)
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &radius.PacketServer{
		Handler:      h,
		SecretSource: radius.StaticSecretSource([]byte(testSecret)),
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.Serve(pc)
	}()
	t.Cleanup(func() {
		_ = srv.Shutdown(context.Background())
		_ = pc.Close()
		<-done
	})
	time.Sleep(20 * time.Millisecond)
	return pc.LocalAddr().String()
}

func sendAccounting(t *testing.T, addr string, statusType rfc2866.AcctStatusType, sessionID, username string, bytesIn, bytesOut uint32) {
	t.Helper()
	pkt := radius.New(radius.CodeAccountingRequest, []byte(testSecret))
	_ = rfc2866.AcctStatusType_Set(pkt, statusType)
	_ = rfc2866.AcctSessionID_SetString(pkt, sessionID)
	_ = rfc2865.UserName_SetString(pkt, username)
	_ = rfc2866.AcctInputOctets_Set(pkt, rfc2866.AcctInputOctets(bytesIn))
	_ = rfc2866.AcctOutputOctets_Set(pkt, rfc2866.AcctOutputOctets(bytesOut))
	_ = rfc2866.AcctSessionTime_Set(pkt, 60)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := radius.Exchange(ctx, pkt, addr)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if resp.Code != radius.CodeAccountingResponse {
		t.Errorf("expected AccountingResponse, got %v", resp.Code)
	}
}

func TestAccounting_Start(t *testing.T) {
	d := openDB(t)
	addr := startServer(t, d)
	sendAccounting(t, addr, rfc2866.AcctStatusType_Value_Start, "sess-1", "alice", 0, 0)

	sessions, err := d.ListActiveSessions()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 active session, got %d", len(sessions))
	}
	if sessions[0].Username != "alice" {
		t.Errorf("expected alice, got %q", sessions[0].Username)
	}
	if sessions[0].Status != "active" {
		t.Errorf("expected active, got %q", sessions[0].Status)
	}
}

func TestAccounting_Interim(t *testing.T) {
	d := openDB(t)
	addr := startServer(t, d)
	sendAccounting(t, addr, rfc2866.AcctStatusType_Value_Start, "sess-2", "bob", 0, 0)
	sendAccounting(t, addr, rfc2866.AcctStatusType_Value_InterimUpdate, "sess-2", "bob", 1024, 2048)

	sessions, err := d.ListActiveSessions()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].BytesIn != 1024 {
		t.Errorf("expected bytes_in 1024, got %d", sessions[0].BytesIn)
	}
	if sessions[0].BytesOut != 2048 {
		t.Errorf("expected bytes_out 2048, got %d", sessions[0].BytesOut)
	}
}

func TestAccounting_Stop(t *testing.T) {
	d := openDB(t)
	addr := startServer(t, d)
	sendAccounting(t, addr, rfc2866.AcctStatusType_Value_Start, "sess-3", "carol", 0, 0)
	sendAccounting(t, addr, rfc2866.AcctStatusType_Value_Stop, "sess-3", "carol", 5000, 10000)

	active, _ := d.ListActiveSessions()
	if len(active) != 0 {
		t.Errorf("expected 0 active sessions after stop, got %d", len(active))
	}
	all, _ := d.ListSessionsByUser("carol")
	if len(all) != 1 {
		t.Fatalf("expected 1 session for carol, got %d", len(all))
	}
	if all[0].Status != "stopped" {
		t.Errorf("expected stopped, got %q", all[0].Status)
	}
	if all[0].BytesIn != 5000 || all[0].BytesOut != 10000 {
		t.Errorf("unexpected bytes: in=%d out=%d", all[0].BytesIn, all[0].BytesOut)
	}
}

func TestAccounting_StartStopCycle(t *testing.T) {
	d := openDB(t)
	addr := startServer(t, d)
	for i, user := range []string{"u1", "u2", "u3"} {
		sid := "sess-" + user
		sendAccounting(t, addr, rfc2866.AcctStatusType_Value_Start, sid, user, 0, 0)
		if i < 2 {
			sendAccounting(t, addr, rfc2866.AcctStatusType_Value_Stop, sid, user, 100, 200)
		}
	}
	active, _ := d.ListActiveSessions()
	if len(active) != 1 {
		t.Errorf("expected 1 active session, got %d", len(active))
	}
	recent, _ := d.ListRecentSessions(10)
	if len(recent) != 3 {
		t.Errorf("expected 3 total sessions, got %d", len(recent))
	}
}

func TestTotalBytes_Gigawords(t *testing.T) {
	// 1 gigaword + 500 octets = 4294967296 + 500
	result := accounting.TotalBytes(500, 1)
	want := int64(4294967796)
	if result != want {
		t.Errorf("expected %d, got %d", want, result)
	}
}
