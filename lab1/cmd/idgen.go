package main

import (
	crand "crypto/rand"
	"encoding/hex"
	mrand "math/rand"
)

const instanceIDBytes = 8

func generateInstanceID() string {
	buf := make([]byte, instanceIDBytes)
	if _, err := crand.Read(buf); err != nil {
		mrand.Read(buf)
	}
	return hex.EncodeToString(buf)
}
