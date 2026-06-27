package auth

import (
	"golang.org/x/crypto/bcrypt"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

func GetPAPPassword(req *radius.Request) string {
	return rfc2865.UserPassword_GetString(req.Packet)
}

func VerifyPAP(req *radius.Request, secret, passwordHash string) bool {
	plain := GetPAPPassword(req)
	if plain == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(plain)) == nil
}
