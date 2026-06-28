package auth

import (
	"crypto/hmac"
	"crypto/md5" //nolint:gosec // HMAC-MD5 required by RFC 3579 §3.2 (Message-Authenticator)
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2869"
	"layeh.com/radius/vendors/microsoft"
	"layeh.com/radius/vendors/wispr"

	"github.com/selvakn/radius-server/internal/db"
)

const mikrotikVendorID = 14988
const mikrotikAttrRateLimit = 8

type Handler struct {
	db     *db.DB
	secret string
}

func New(database *db.DB, secret string) *Handler {
	return &Handler{db: database, secret: secret}
}

func (h *Handler) ServeRADIUS(w radius.ResponseWriter, r *radius.Request) {
	username := rfc2865.UserName_GetString(r.Packet)

	user, err := h.db.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			slog.Info("radius reject: user not found", "username", username)
			h.record(username, "rejected", GetPAPPassword(r))
			h.writeResponse(w, r.Response(radius.CodeAccessReject))
			return
		}
		slog.Error("radius db error", "err", err)
		h.writeResponse(w, r.Response(radius.CodeAccessReject))
		return
	}

	if !user.Enabled {
		slog.Info("radius reject: user disabled", "username", username)
		h.record(username, "rejected", "")
		h.writeResponse(w, r.Response(radius.CodeAccessReject))
		return
	}

	if IsMSCHAP2Request(r.Packet) {
		h.handleMSCHAP2(w, r, user)
		return
	}

	// PAP fallback
	attempted := GetPAPPassword(r)
	slog.Debug("radius pap attempt", "username", username, "has_password", attempted != "")
	if !VerifyPAP(r, h.secret, user.PasswordHash) {
		slog.Info("radius reject: wrong password", "username", username)
		h.record(username, "rejected", attempted)
		h.writeResponse(w, r.Response(radius.CodeAccessReject))
		return
	}

	slog.Info("radius accept (pap)", "username", username)
	h.record(username, "accepted", "")
	resp := r.Response(radius.CodeAccessAccept)
	_ = rfc2869.AcctInterimInterval_Set(resp, 300)
	if user.DownloadRate != nil && user.UploadRate != nil {
		addBandwidthAttributes(resp, *user.DownloadRate, *user.UploadRate)
	}
	h.writeResponse(w, resp)
}

func (h *Handler) handleMSCHAP2(w radius.ResponseWriter, r *radius.Request, user *db.User) {
	username := user.Username

	if user.NTHash == "" {
		slog.Info("radius reject: no nt_hash stored (re-save password in admin UI)", "username", username)
		h.record(username, "rejected", "")
		h.writeResponse(w, r.Response(radius.CodeAccessReject))
		return
	}

	ntResponse, ok := VerifyMSCHAP2(r, username, user.NTHash)
	if !ok {
		slog.Info("radius reject: ms-chapv2 wrong password", "username", username)
		h.record(username, "rejected", "")
		h.writeResponse(w, r.Response(radius.CodeAccessReject))
		return
	}

	slog.Info("radius accept (ms-chapv2)", "username", username)
	h.record(username, "accepted", "")
	resp := r.Response(radius.CodeAccessAccept)
	_ = rfc2869.AcctInterimInterval_Set(resp, 300)
	if success := MSCHAPv2Success(r, username, user.NTHash, ntResponse); success != nil {
		_ = microsoft.MSCHAP2Success_Add(resp, success)
	}
	if user.DownloadRate != nil && user.UploadRate != nil {
		addBandwidthAttributes(resp, *user.DownloadRate, *user.UploadRate)
	}
	h.writeResponse(w, resp)
}

func (h *Handler) record(username, outcome, password string) {
	if err := h.db.RecordAttempt(username, outcome, password); err != nil {
		slog.Error("record attempt", "err", err)
	}
}

// writeResponse adds Message-Authenticator then sends the packet.
func (h *Handler) writeResponse(w radius.ResponseWriter, resp *radius.Packet) {
	addMessageAuthenticator(resp, h.secret)
	_ = w.Write(resp)
}

// addMessageAuthenticator computes and sets RFC 3579 Message-Authenticator.
// Flow: set MA=zeros, MarshalBinary (preserves RequestAuth), HMAC-MD5 over
// those bytes, set real MA — Encode() then computes Response-Auth correctly.
func addMessageAuthenticator(p *radius.Packet, secret string) {
	_ = rfc2869.MessageAuthenticator_Set(p, make([]byte, 16))
	raw, err := p.MarshalBinary()
	if err != nil {
		return
	}
	mac := hmac.New(md5.New, []byte(secret))
	mac.Write(raw)
	_ = rfc2869.MessageAuthenticator_Set(p, mac.Sum(nil))
}

// addBandwidthAttributes adds both MikroTik and WISPr bandwidth VSAs.
func addBandwidthAttributes(p *radius.Packet, downKbps, upKbps int) {
	// MikroTik Rate-Limit (vendor 14988, attr 8): string "Dk/Uk"
	vsa := mikrotikRateLimit(downKbps, upKbps)
	p.Add(radius.Type(26), vsa)

	// WISPr-Bandwidth-Max-Down / Up: uint32 bits/sec (rates bounded 1–500 Mbps, no overflow)
	bitsDown := wispr.WISPrBandwidthMaxDown(downKbps) * 1000 //nolint:gosec
	bitsUp := wispr.WISPrBandwidthMaxUp(upKbps) * 1000       //nolint:gosec
	_ = wispr.WISPrBandwidthMaxDown_Set(p, bitsDown)
	_ = wispr.WISPrBandwidthMaxUp_Set(p, bitsUp)
}

func mikrotikRateLimit(downKbps, upKbps int) radius.Attribute {
	rateStr := fmt.Sprintf("%dk/%dk", downKbps, upKbps)
	value := []byte(rateStr)

	// VSA layout: 4 bytes vendor-ID, 1 byte vendor-type, 1 byte vendor-length, N bytes value
	vsa := make([]byte, 6+len(value))
	binary.BigEndian.PutUint32(vsa[0:], mikrotikVendorID)
	vsa[4] = mikrotikAttrRateLimit
	vsa[5] = byte(2 + len(value)) //nolint:gosec // bounded: rate string max ~15 bytes
	copy(vsa[6:], value)

	return radius.Attribute(vsa)
}
