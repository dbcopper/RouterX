package util

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func HashString(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])
}

func NormalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
