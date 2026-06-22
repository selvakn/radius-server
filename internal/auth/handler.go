package auth

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"

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
			_ = w.Write(r.Response(radius.CodeAccessReject))
			return
		}
		slog.Error("radius db error", "err", err)
		_ = w.Write(r.Response(radius.CodeAccessReject))
		return
	}

	if !user.Enabled {
		slog.Info("radius reject: user disabled", "username", username)
		_ = w.Write(r.Response(radius.CodeAccessReject))
		return
	}

	if !VerifyPAP(r, h.secret, user.PasswordHash) {
		slog.Info("radius reject: wrong password", "username", username)
		_ = w.Write(r.Response(radius.CodeAccessReject))
		return
	}

	slog.Info("radius accept", "username", username)
	resp := r.Response(radius.CodeAccessAccept)
	_ = rfc2865.FramedProtocol_Set(resp, rfc2865.FramedProtocol_Value_PPP)

	if user.DownloadRate != nil && user.UploadRate != nil {
		vsa := mikrotikRateLimit(*user.DownloadRate, *user.UploadRate)
		resp.Add(radius.Type(26), vsa)
	}

	_ = w.Write(resp)
}

func mikrotikRateLimit(downKbps, upKbps int) radius.Attribute {
	rateStr := fmt.Sprintf("%dk/%dk", downKbps, upKbps)
	value := []byte(rateStr)

	attrLen := 2 + len(value)
	vsa := make([]byte, 4+2+attrLen)
	binary.BigEndian.PutUint32(vsa[0:], mikrotikVendorID)
	vsa[4] = mikrotikAttrRateLimit
	vsa[5] = byte(attrLen)
	copy(vsa[6:], value)

	return radius.Attribute(vsa)
}
