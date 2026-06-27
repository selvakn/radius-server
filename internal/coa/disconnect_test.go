package coa_test

import (
	"context"
	"net"
	"testing"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2866"

	"github.com/selvakn/radius-server/internal/coa"
)

const testSecret = "testsecret"

func startMockNAS(t *testing.T, responseCode radius.Code) string {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := pc.LocalAddr().String()
	go func() {
		defer func() { _ = pc.Close() }()
		buf := make([]byte, 4096)
		for {
			n, remoteAddr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			req, err := radius.Parse(buf[:n], []byte(testSecret))
			if err != nil {
				continue
			}
			resp := req.Response(responseCode)
			encoded, _ := resp.Encode()
			_, _ = pc.WriteTo(encoded, remoteAddr)
		}
	}()
	t.Cleanup(func() { _ = pc.Close() })
	time.Sleep(10 * time.Millisecond)
	return addr
}

func TestSendDisconnect_ACK(t *testing.T) {
	addr := startMockNAS(t, radius.CodeDisconnectACK)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := coa.SendDisconnect(ctx, addr, testSecret, "sess-1", "alice"); err != nil {
		t.Errorf("expected nil error on ACK, got: %v", err)
	}
}

func TestSendDisconnect_NAK(t *testing.T) {
	addr := startMockNAS(t, radius.CodeDisconnectNAK)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := coa.SendDisconnect(ctx, addr, testSecret, "sess-2", "bob"); err == nil {
		t.Error("expected error on NAK, got nil")
	}
}

func TestSendDisconnect_Timeout(t *testing.T) {
	// Nothing listening on this port → timeout
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	err := coa.SendDisconnect(ctx, "127.0.0.1:19999", testSecret, "sess-3", "carol")
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestSendDisconnect_IncludesSessionAttributes(t *testing.T) {
	var received *radius.Packet
	pc, _ := net.ListenPacket("udp", "127.0.0.1:0")
	addr := pc.LocalAddr().String()
	go func() {
		defer func() { _ = pc.Close() }()
		buf := make([]byte, 4096)
		n, remoteAddr, _ := pc.ReadFrom(buf)
		pkt, _ := radius.Parse(buf[:n], []byte(testSecret))
		received = pkt
		resp := pkt.Response(radius.CodeDisconnectACK)
		encoded, _ := resp.Encode()
		_, _ = pc.WriteTo(encoded, remoteAddr)
	}()
	time.Sleep(10 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = coa.SendDisconnect(ctx, addr, testSecret, "my-session-id", "myuser")

	if received == nil {
		t.Fatal("no packet received")
	}
	if rfc2866.AcctSessionID_GetString(received) != "my-session-id" {
		t.Errorf("expected AcctSessionID 'my-session-id', got %q", rfc2866.AcctSessionID_GetString(received))
	}
}
