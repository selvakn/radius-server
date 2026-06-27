package auth

import (
	"log/slog"

	"golang.org/x/crypto/bcrypt"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

func GetPAPPassword(req *radius.Request) string {
	plain, err := rfc2865.UserPassword_LookupString(req.Packet)
	if err != nil {
		slog.Debug("pap: could not extract User-Password", "err", err)
		return ""
	}
	return plain
}

func VerifyPAP(req *radius.Request, secret, passwordHash string) bool {
	plain := GetPAPPassword(req)
	if plain == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(plain)) == nil
}
