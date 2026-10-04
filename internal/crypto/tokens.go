package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

// Hash takes a string and returns its sha256 hash as a hex encoded string
func Hash(string string) string {
	hashArray := sha256.Sum256([]byte(string))
	return hex.EncodeToString(hashArray[:])
}
