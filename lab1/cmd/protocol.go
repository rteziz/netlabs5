package main

import (
	"encoding/hex"
	"strings"
)

const (
	protocolMagic       = "MCD2"
	instanceIDHexLen    = instanceIDBytes * 2
	maxHostnameLen      = 253
	heartbeatHeaderSize = len(protocolMagic) + instanceIDHexLen + 1
	maxHeartbeatSize    = heartbeatHeaderSize + maxHostnameLen
)

type heartbeat struct {
	ID       string
	Hostname string
}

func buildHeartbeat(id, hostname string) []byte {
	data := []byte(protocolMagic + id)
	data = append(data, byte(len(hostname)))
	return append(data, hostname...)
}

func parseHeartbeat(data []byte) (heartbeat, bool) {
	if len(data) < heartbeatHeaderSize || len(data) > maxHeartbeatSize {
		return heartbeat{}, false
	}
	if string(data[:len(protocolMagic)]) != protocolMagic {
		return heartbeat{}, false
	}
	id := string(data[len(protocolMagic) : len(protocolMagic)+instanceIDHexLen])
	hostname := string(data[heartbeatHeaderSize:])
	if int(data[heartbeatHeaderSize-1]) != len(hostname) || !isValidInstanceID(id) || !isValidHostname(hostname) {
		return heartbeat{}, false
	}
	return heartbeat{ID: strings.ToLower(id), Hostname: hostname}, true
}

func isValidInstanceID(id string) bool {
	if len(id) != instanceIDHexLen {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func truncateHostname(hostname string) string {
	labels := strings.Split(hostname, ".")
	for i, label := range labels {
		if len(label) > 63 {
			labels[i] = strings.TrimRight(label[:63], "-")
		}
	}
	hostname = strings.Join(labels, ".")
	if len(hostname) > maxHostnameLen {
		hostname = strings.TrimRight(hostname[:maxHostnameLen], ".-")
	}
	return hostname
}

// Hostname consists of ASCII labels separated by dots, with no control characters.
func isValidHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > maxHostnameLen {
		return false
	}
	for _, label := range strings.Split(hostname, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}
