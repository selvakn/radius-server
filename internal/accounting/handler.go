package accounting

import (
	"log/slog"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
	"layeh.com/radius/rfc2869"

	"github.com/selvakn/radius-server/internal/db"
)

type Handler struct {
	db *db.DB
}

func New(database *db.DB) *Handler {
	return &Handler{db: database}
}

func (h *Handler) ServeRADIUS(w radius.ResponseWriter, r *radius.Request) {
	statusType := rfc2866.AcctStatusType_Get(r.Packet)
	sessionID := rfc2866.AcctSessionID_GetString(r.Packet)
	username := rfc2865.UserName_GetString(r.Packet)
	nasIP := rfc2865.NASIPAddress_Get(r.Packet).String()

	bytesIn := TotalBytes(
		int64(rfc2866.AcctInputOctets_Get(r.Packet)),
		int64(rfc2869.AcctInputGigawords_Get(r.Packet)),
	)
	bytesOut := TotalBytes(
		int64(rfc2866.AcctOutputOctets_Get(r.Packet)),
		int64(rfc2869.AcctOutputGigawords_Get(r.Packet)),
	)
	sessionTime := int64(rfc2866.AcctSessionTime_Get(r.Packet))

	switch statusType {
	case rfc2866.AcctStatusType_Value_Start:
		if err := h.db.UpsertSessionStart(sessionID, username, nasIP, time.Now()); err != nil {
			slog.Error("accounting start", "err", err, "session", sessionID)
		} else {
			slog.Info("session start", "user", username, "session", sessionID)
		}

	case rfc2866.AcctStatusType_Value_InterimUpdate:
		if err := h.db.UpdateSessionInterim(sessionID, bytesIn, bytesOut, sessionTime); err != nil {
			slog.Error("accounting interim", "err", err, "session", sessionID)
		} else {
			slog.Info("session interim", "user", username, "in", bytesIn, "out", bytesOut)
		}

	case rfc2866.AcctStatusType_Value_Stop:
		cause := rfc2866.AcctTerminateCause_Get(r.Packet).String()
		if err := h.db.StopSession(sessionID, bytesIn, bytesOut, sessionTime, cause, time.Now()); err != nil {
			slog.Error("accounting stop", "err", err, "session", sessionID)
		} else {
			slog.Info("session stop", "user", username, "in", bytesIn, "out", bytesOut, "cause", cause)
		}

	case rfc2866.AcctStatusType_Value_AccountingOn,
		rfc2866.AcctStatusType_Value_AccountingOff:
		slog.Info("accounting power cycle", "type", statusType, "nas", nasIP)

	default:
		slog.Warn("unknown accounting status type", "type", statusType)
	}

	_ = w.Write(r.Response(radius.CodeAccountingResponse))
}

// TotalBytes combines 32-bit octets and gigawords into a full int64 byte count.
func TotalBytes(octets, gigawords int64) int64 {
	return gigawords*4294967296 + octets
}
