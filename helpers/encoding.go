package helpers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

func GenerateToken() (string, string, error) {
	rawToken := make([]byte, 32)
	_, err := rand.Read(rawToken)
	if err != nil {
		return "", "", err
	}
	confirmToken := base64.RawURLEncoding.EncodeToString(rawToken)

	rawConfirmTokenHash := sha256.Sum256([]byte(confirmToken))
	confirmTokenHash := hex.EncodeToString(rawConfirmTokenHash[:])

	return confirmTokenHash, string(rawToken), nil
}
