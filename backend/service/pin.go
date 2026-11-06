package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

// O PIN tem so 4 digitos, entao o hash serve para nao guardar o valor em
// texto puro, nao para resistir a forca bruta. A protecao real vem do
// relogio exigir a tag fisica junto com o PIN.
func hashPin(pin string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	saltHex := hex.EncodeToString(salt)

	return saltHex + "$" + digestPin(saltHex, pin), nil
}

func verifyPin(pinHash, pin string) bool {
	salt, digest, ok := strings.Cut(pinHash, "$")
	if !ok {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(digest), []byte(digestPin(salt, pin))) == 1
}

func digestPin(salt, pin string) string {
	sum := sha256.Sum256([]byte(salt + ":" + pin))
	return hex.EncodeToString(sum[:])
}
