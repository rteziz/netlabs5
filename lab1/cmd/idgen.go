package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

const instanceIDBytes = 8

func generateInstanceID() (string, error) {
	return readInstanceID(rand.Reader)
}

func readInstanceID(source io.Reader) (string, error) {
	buf := make([]byte, instanceIDBytes)
	if _, err := io.ReadFull(source, buf); err != nil {
		return "", fmt.Errorf("не удалось создать уникальный ID: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
