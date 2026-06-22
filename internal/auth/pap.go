package auth

import (
	"golang.org/x/crypto/bcrypt"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

func VerifyPAP(req *radius.Request, secret, passwordHash string) bool {
	plain := rfc2865.UserPassword_GetString(req.Packet)
	if plain == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(plain)) == nil
}
