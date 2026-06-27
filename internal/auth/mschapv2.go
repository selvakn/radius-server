package auth

import (
	"bytes"
	"crypto/sha1" //nolint:gosec // SHA1 required by MS-CHAPv2 RFC 2759 §8.7
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"layeh.com/radius"
	"layeh.com/radius/rfc2759"
	"layeh.com/radius/vendors/microsoft"

	"golang.org/x/crypto/md4" //nolint:staticcheck,gosec // MD4 required by MS-CHAPv2 (RFC 2759)
)

// RFC 2759 §8.7 magic constants
var (
	mschap2Magic1 = []byte{
		0x4D, 0x61, 0x67, 0x69, 0x63, 0x20, 0x73, 0x65, 0x72, 0x76,
		0x65, 0x72, 0x20, 0x74, 0x6F, 0x20, 0x63, 0x6C, 0x69, 0x65,
		0x6E, 0x74, 0x20, 0x73, 0x69, 0x67, 0x6E, 0x69, 0x6E, 0x67,
		0x20, 0x63, 0x6F, 0x6E, 0x73, 0x74, 0x61, 0x6E, 0x74,
	}
	mschap2Magic2 = []byte{
		0x50, 0x61, 0x64, 0x20, 0x74, 0x6F, 0x20, 0x6D, 0x61, 0x6B,
		0x65, 0x20, 0x69, 0x74, 0x20, 0x64, 0x6F, 0x20, 0x6D, 0x6F,
		0x72, 0x65, 0x20, 0x74, 0x68, 0x61, 0x6E, 0x20, 0x6F, 0x6E,
		0x65, 0x20, 0x69, 0x74, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6F,
		0x6E,
	}
)

// NTHash computes the NT hash (MD4 of UTF-16LE password) and returns it hex-encoded.
func NTHash(password string) (string, error) {
	utf16, err := rfc2759.ToUTF16([]byte(password))
	if err != nil {
		return "", fmt.Errorf("utf16 encode: %w", err)
	}
	h := md4.New() //nolint:gosec // MD4 required by MS-CHAPv2 (RFC 2759)
	h.Write(utf16)
	return hex.EncodeToString(h.Sum(nil)), nil
}

// IsMSCHAP2Request returns true if the packet contains MS-CHAPv2 attributes.
func IsMSCHAP2Request(p *radius.Packet) bool {
	return len(microsoft.MSCHAPChallenge_Get(p)) == 16 &&
		len(microsoft.MSCHAP2Response_Get(p)) == 50
}

// VerifyMSCHAP2 verifies an MS-CHAPv2 Access-Request using the stored NT hash.
// Returns the NT-Response bytes (needed for Access-Accept) on success, nil on failure.
func VerifyMSCHAP2(req *radius.Request, username, ntHashHex string) (ntResponse []byte, ok bool) {
	authChallenge := microsoft.MSCHAPChallenge_Get(req.Packet)
	response := microsoft.MSCHAP2Response_Get(req.Packet)

	if len(authChallenge) != 16 || len(response) != 50 {
		slog.Debug("mschapv2: missing or malformed attributes",
			"challenge_len", len(authChallenge), "response_len", len(response))
		return nil, false
	}

	ntHash, err := hex.DecodeString(ntHashHex)
	if err != nil || len(ntHash) != 16 {
		slog.Debug("mschapv2: invalid or missing nt_hash for user", "username", username)
		return nil, false
	}

	peerChallenge := response[2:18]
	peerResponse := response[26:50]

	// ChallengeHash = SHA1(PeerChallenge + AuthChallenge + UserName)[:8]
	challenge := rfc2759.ChallengeHash(peerChallenge, authChallenge, []byte(username))
	// Use stored NT hash directly in ChallengeResponse (skips NTPasswordHash step)
	expected := rfc2759.ChallengeResponse(challenge, ntHash)

	if !bytes.Equal(expected, peerResponse) {
		return nil, false
	}
	return expected, true
}

// MSCHAPv2Success builds the MS-CHAP2-Success value for the Access-Accept.
// Implements RFC 2759 §8.7 using stored NT hash instead of plaintext password.
func MSCHAPv2Success(req *radius.Request, username, ntHashHex string, ntResponse []byte) []byte {
	response := microsoft.MSCHAP2Response_Get(req.Packet)
	authChallenge := microsoft.MSCHAPChallenge_Get(req.Packet)
	if len(response) < 18 || len(authChallenge) != 16 {
		return nil
	}
	ident := response[0]
	peerChallenge := response[2:18]

	ntHash, err := hex.DecodeString(ntHashHex)
	if err != nil {
		return nil
	}

	// hashHash = MD4(ntHash)
	hh := md4.New() //nolint:gosec
	hh.Write(ntHash)
	hashHash := hh.Sum(nil)

	// SHA1(hashHash + ntResponse + magic1)
	s := sha1.New() //nolint:gosec
	s.Write(hashHash)
	s.Write(ntResponse)
	s.Write(mschap2Magic1)
	digest := s.Sum(nil)

	// challenge = ChallengeHash(peerChallenge, authChallenge, username)[:8]
	challengeHash := rfc2759.ChallengeHash(peerChallenge, authChallenge, []byte(username))

	// SHA1(digest + challenge + magic2)
	s = sha1.New() //nolint:gosec
	s.Write(digest)
	s.Write(challengeHash)
	s.Write(mschap2Magic2)
	digest = s.Sum(nil)

	authResp := "S=" + strings.ToUpper(hex.EncodeToString(digest))

	// MS-CHAP2-Success: Ident(1) + authResp (42 bytes) = 43 bytes
	success := make([]byte, 43)
	success[0] = ident
	copy(success[1:], []byte(authResp))
	return success
}
