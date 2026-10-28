package main

import (
	"filetransfer/protocol"
	"net"
	"strings"
	"testing"
	"time"
)

func TestFinalStatus(t *testing.T) {
	for _, tc := range []struct {
		code byte
		want string
	}{
		{protocol.StatusOK, ""},
		{protocol.StatusFail, "server reported a failure"},
		{protocol.StatusReady, "unexpected final status"},
	} {
		a, b := net.Pipe()
		a.SetDeadline(time.Now().Add(time.Second))
		go func() { defer b.Close(); b.Write([]byte{tc.code}) }()
		err := waitForStatus(a)
		a.Close()
		if tc.want == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("status %d: %v", tc.code, err)
		}
	}
}

func TestAdmissionAndFinalStatus(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	a.SetDeadline(time.Now().Add(time.Second))
	go func() { defer b.Close(); b.Write([]byte{protocol.StatusReady, protocol.StatusOK}) }()
	if err := waitForAdmission(a); err != nil {
		t.Fatal(err)
	}
	if err := waitForStatus(a); err != nil {
		t.Fatal(err)
	}
}
