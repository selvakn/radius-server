package auth_test

import (
	"bytes"
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2869"
	"layeh.com/radius/vendors/wispr"

	"github.com/selvakn/radius-server/internal/auth"
	"github.com/selvakn/radius-server/internal/db"
)

func openDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func createUser(t *testing.T, d *db.DB, username, password string, enabled bool, down, up *int) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	if err := d.CreateUser(db.User{
		Username:     username,
		PasswordHash: string(hash),
		Enabled:      enabled,
		DownloadRate: down,
		UploadRate:   up,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func sendRADIUS(t *testing.T, addr, secret, username, password string) *radius.Packet {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pkt := radius.New(radius.CodeAccessRequest, []byte(secret))
	_ = rfc2865.UserName_SetString(pkt, username)
	_ = rfc2865.UserPassword_SetString(pkt, password)
	resp, err := radius.Exchange(ctx, pkt, addr)
	if err != nil {
		t.Fatalf("radius exchange: %v", err)
	}
	return resp
}

func startServer(t *testing.T, d *db.DB) string {
	t.Helper()
	h := auth.New(d, testSecret)
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := pc.LocalAddr().String()
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
	return addr
}

func TestHandler_AcceptValidUser(t *testing.T) {
	d := openDB(t)
	createUser(t, d, "alice", "pass123", true, nil, nil)
	addr := startServer(t, d)
	resp := sendRADIUS(t, addr, testSecret, "alice", "pass123")
	if resp.Code != radius.CodeAccessAccept {
		t.Errorf("expected Accept, got %v", resp.Code)
	}
}

func TestHandler_RejectUnknownUser(t *testing.T) {
	d := openDB(t)
	addr := startServer(t, d)
	resp := sendRADIUS(t, addr, testSecret, "nobody", "pass")
	if resp.Code != radius.CodeAccessReject {
		t.Errorf("expected Reject, got %v", resp.Code)
	}
}

func TestHandler_RejectDisabledUser(t *testing.T) {
	d := openDB(t)
	createUser(t, d, "disabled", "pass", false, nil, nil)
	addr := startServer(t, d)
	resp := sendRADIUS(t, addr, testSecret, "disabled", "pass")
	if resp.Code != radius.CodeAccessReject {
		t.Errorf("expected Reject for disabled user, got %v", resp.Code)
	}
}

func TestHandler_RejectWrongPassword(t *testing.T) {
	d := openDB(t)
	createUser(t, d, "bob", "correct", true, nil, nil)
	addr := startServer(t, d)
	resp := sendRADIUS(t, addr, testSecret, "bob", "wrong")
	if resp.Code != radius.CodeAccessReject {
		t.Errorf("expected Reject for wrong password, got %v", resp.Code)
	}
}

func TestHandler_IncludesRateLimitVSA(t *testing.T) {
	d := openDB(t)
	down, up := 2048, 1024
	createUser(t, d, "limited", "pass", true, &down, &up)
	addr := startServer(t, d)
	resp := sendRADIUS(t, addr, testSecret, "limited", "pass")
	if resp.Code != radius.CodeAccessAccept {
		t.Errorf("expected Accept, got %v", resp.Code)
	}
	// MikroTik VSA
	vsa := []byte(resp.Get(radius.Type(26)))
	rateStr := "2048k/1024k"
	wantLen := 6 + len(rateStr)
	if len(vsa) != wantLen {
		t.Errorf("MikroTik VSA length: want %d, got %d (raw: %x)", wantLen, len(vsa), vsa)
	}
	if !bytes.Contains(vsa, []byte(rateStr)) {
		t.Errorf("expected MikroTik rate limit %q in VSA, got: %x", rateStr, vsa)
	}
	// WISPr bandwidth
	if wispr.WISPrBandwidthMaxDown_Get(resp) != wispr.WISPrBandwidthMaxDown(2048000) {
		t.Errorf("expected WISPr down 2048000 bps, got %d", wispr.WISPrBandwidthMaxDown_Get(resp))
	}
	if wispr.WISPrBandwidthMaxUp_Get(resp) != wispr.WISPrBandwidthMaxUp(1024000) {
		t.Errorf("expected WISPr up 1024000 bps, got %d", wispr.WISPrBandwidthMaxUp_Get(resp))
	}
}

func TestHandler_IncludesMessageAuthenticator(t *testing.T) {
	d := openDB(t)
	createUser(t, d, "mauser", "pass", true, nil, nil)
	addr := startServer(t, d)

	resp := sendRADIUS(t, addr, testSecret, "mauser", "pass")
	if resp.Code != radius.CodeAccessAccept {
		t.Fatalf("expected Accept, got %v", resp.Code)
	}
	ma := rfc2869.MessageAuthenticator_Get(resp)
	if len(ma) != 16 {
		t.Errorf("expected 16-byte Message-Authenticator, got %d bytes", len(ma))
	}

	resp2 := sendRADIUS(t, addr, testSecret, "mauser", "wrongpass")
	if resp2.Code != radius.CodeAccessReject {
		t.Fatalf("expected Reject, got %v", resp2.Code)
	}
	ma2 := rfc2869.MessageAuthenticator_Get(resp2)
	if len(ma2) != 16 {
		t.Errorf("expected 16-byte Message-Authenticator on Reject, got %d bytes", len(ma2))
	}
}

func TestHandler_RecordsAcceptAttempt(t *testing.T) {
	d := openDB(t)
	createUser(t, d, "recorded", "pass", true, nil, nil)
	addr := startServer(t, d)
	sendRADIUS(t, addr, testSecret, "recorded", "pass")

	summaries, err := d.ListAttemptSummaries()
	if err != nil {
		t.Fatalf("list summaries: %v", err)
	}
	found := false
	for _, s := range summaries {
		if s.Username == "recorded" && s.LastOutcome == "accepted" {
			found = true
		}
	}
	if !found {
		t.Error("expected accepted attempt to be recorded")
	}
}

func TestHandler_RecordsRejectAttempt(t *testing.T) {
	d := openDB(t)
	addr := startServer(t, d)
	sendRADIUS(t, addr, testSecret, "nobody", "wrongpass")

	summaries, err := d.ListAttemptSummaries()
	if err != nil {
		t.Fatalf("list summaries: %v", err)
	}
	found := false
	for _, s := range summaries {
		if s.Username == "nobody" && s.LastOutcome == "rejected" {
			found = true
			if s.LastPassword != "wrongpass" {
				t.Errorf("expected LastPassword 'wrongpass', got %q", s.LastPassword)
			}
		}
	}
	if !found {
		t.Error("expected rejected attempt to be recorded")
	}
}
