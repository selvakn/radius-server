package auth_test

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"

	"github.com/selvakn/radius-server/internal/auth"
)

const testSecret = "testsecret"

func makeRequest(t *testing.T, username, password string) *radius.Request {
	t.Helper()
	pkt := radius.New(radius.CodeAccessRequest, []byte(testSecret))
	rfc2865.UserName_SetString(pkt, username)
	if err := rfc2865.UserPassword_SetString(pkt, password); err != nil {
		t.Fatalf("set password: %v", err)
	}
	return &radius.Request{Packet: pkt}
}

func hashPassword(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	return string(h)
}

func TestVerifyPAP_ValidPassword(t *testing.T) {
	req := makeRequest(t, "alice", "correcthorse")
	hash := hashPassword(t, "correcthorse")
	if !auth.VerifyPAP(req, testSecret, hash) {
		t.Error("expected PAP to return true for correct password")
	}
}

func TestVerifyPAP_WrongPassword(t *testing.T) {
	req := makeRequest(t, "alice", "wrongpassword")
	hash := hashPassword(t, "correcthorse")
	if auth.VerifyPAP(req, testSecret, hash) {
		t.Error("expected PAP to return false for wrong password")
	}
}

func TestVerifyPAP_NoPasswordAttribute(t *testing.T) {
	pkt := radius.New(radius.CodeAccessRequest, []byte(testSecret))
	rfc2865.UserName_SetString(pkt, "alice")
	req := &radius.Request{Packet: pkt}
	if auth.VerifyPAP(req, testSecret, hashPassword(t, "pass")) {
		t.Error("expected false when no User-Password attribute")
	}
}
