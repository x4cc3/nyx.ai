package ids

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

func New(prefix string) string {
	var buf [6]byte
	_, _ = rand.Read(buf[:])
	return strings.ToLower(prefix) + "_" + hex.EncodeToString(buf[:])
}
