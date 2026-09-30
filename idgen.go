package main

import (
	"crypto/rand"
	"encoding/hex"
)

func generateInstanceID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "fallback-instance-id"
	}
	return hex.EncodeToString(buf)
}
