package main

import (
	"encoding/hex"
)

const (
	protocolMagic    = "MCD1"
	instanceIDHexLen = instanceIDBytes * 2
	heartbeatSize    = len(protocolMagic) + instanceIDHexLen
)

func buildHeartbeat(id string) []byte {
	return []byte(protocolMagic + id)
}

func parseHeartbeat(data []byte) (string, bool) {
	if len(data) != heartbeatSize {
		return "", false
	}
	if string(data[:len(protocolMagic)]) != protocolMagic {
		return "", false
	}
	id := string(data[len(protocolMagic):])
	if !isValidInstanceID(id) {
		return "", false
	}
	return id, true
}

func isValidInstanceID(id string) bool {
	if len(id) != instanceIDHexLen {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}
