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
	vsas := resp.Get(radius.Type(26))
	if vsas == nil {
		t.Fatal("expected VSA attribute in response")
	}
	if !bytes.Contains([]byte(vsas), []byte("2048k/1024k")) {
		t.Errorf("expected rate limit '2048k/1024k' in VSA, got: %q", vsas)
	}
}
