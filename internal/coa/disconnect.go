package coa

import (
	"context"
	"fmt"
	"strings"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
)

// SendDisconnect sends an RFC 5176 Disconnect-Request to the NAS and returns
// nil on Disconnect-ACK, or an error on Disconnect-NAK or exchange failure.
// nasAddr may be a bare IP ("192.168.1.1") or IP:port; bare IPs get port 3799 appended.
// The caller is responsible for setting a deadline on ctx (5 seconds recommended).
func SendDisconnect(ctx context.Context, nasAddr, secret, sessionID, username string) error {
	pkt := radius.New(radius.CodeDisconnectRequest, []byte(secret))
	_ = rfc2865.UserName_SetString(pkt, username)
	_ = rfc2866.AcctSessionID_SetString(pkt, sessionID)

	addr := nasAddr
	if !strings.Contains(nasAddr, ":") {
		addr = nasAddr + ":3799"
	}

	resp, err := radius.Exchange(ctx, pkt, addr)
	if err != nil {
		return fmt.Errorf("disconnect request failed: %w", err)
	}
	if resp.Code != radius.CodeDisconnectACK {
		return fmt.Errorf("NAS rejected disconnect (code %v)", resp.Code)
	}
	return nil
}
