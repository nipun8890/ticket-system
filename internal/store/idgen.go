package store

import (
	"crypto/rand"
	"encoding/hex"
)

// newID generates a random hex identifier prefixed with the given string,
// e.g. "usr_3f9a1b2c...". Uses crypto/rand only, so no external UUID
// library is required.
func newID(prefix string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}
