package session

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SignID returns id with an HMAC-SHA256 signature appended, keyed by
// secret, so a client can't forge or guess another session's ID. The
// registry itself is still keyed by the raw id — only the token handed
// back to the client (and expected from it afterward) is signed.
func SignID(secret, id string) string {
	return id + "." + hex.EncodeToString(mac(secret, id))
}

// VerifyID checks a token produced by SignID and, if the signature
// matches, returns the raw id it was signed for.
func VerifyID(secret, token string) (string, bool) {
	i := strings.LastIndexByte(token, '.')
	if i < 0 {
		return "", false
	}
	id, sig := token[:i], token[i+1:]

	got, err := hex.DecodeString(sig)
	if err != nil {
		return "", false
	}
	if !hmac.Equal(got, mac(secret, id)) {
		return "", false
	}
	return id, true
}

func mac(secret, id string) []byte {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(id))
	return h.Sum(nil)
}
