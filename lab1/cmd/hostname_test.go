package main

import (
	"strings"
	"testing"
)

func TestTruncateHostname(t *testing.T) {
	a63 := strings.Repeat("a", 63)
	prefix := strings.Repeat(a63+".", 3)
	for _, tc := range []struct{ name, input, want string }{
		{"unchanged", "my-pc.local", "my-pc.local"},
		{"long label", strings.Repeat("a", 300), a63},
		{"each label", strings.Repeat("a", 70) + ".local", a63 + ".local"},
		{"label ends in hyphen", strings.Repeat("a", 62) + "-bc", strings.Repeat("a", 62)},
		{"total limit", prefix + a63, prefix + strings.Repeat("a", 61)},
		{"total cut at dot", prefix + strings.Repeat("b", 60) + ".extra", prefix + strings.Repeat("b", 60)},
		{"total cut at hyphen", prefix + strings.Repeat("b", 60) + "-z", prefix + strings.Repeat("b", 60)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateHostname(tc.input)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			id := "0123456789abcdef"
			packet, ok := parseHeartbeat(buildHeartbeat(id, got))
			if !ok || packet.Hostname != got || packet.ID != id {
				t.Fatal("shortened hostname failed round trip")
			}
		})
	}
}
