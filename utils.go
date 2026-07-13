package main

import (
	"crypto/rand"
	"encoding/hex"
)

// generateID creates a random 16-character hex string to be used as a unique ID.
func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
