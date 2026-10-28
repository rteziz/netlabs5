package main

import (
	"bytes"
	"log"
	"net"
	"strings"
	"testing"
	"time"
)

func TestMeterFinalAfterPeriodic(t *testing.T) {
	var out bytes.Buffer
	old := log.Writer()
	log.SetOutput(&out)
	defer log.SetOutput(old)
	m, finish := startMeter("test")
	m.Write([]byte("abc"))
	time.Sleep(ReportInterval + 100*time.Millisecond)
	m.Write([]byte("def"))
	finish()
	if !strings.Contains(out.String(), "periodic:") || !strings.Contains(out.String(), "final:") {
		t.Fatal(out.String())
	}
	if m.total.Load() != 6 {
		t.Fatal("wrong byte count")
	}
}

func TestHeaderFailureReportsSpeed(t *testing.T) {
	var out bytes.Buffer
	old := log.Writer()
	log.SetOutput(&out)
	defer log.SetOutput(old)
	a, b := net.Pipe()
	defer a.Close()
	b.Close()
	if err := receiveFile(a, "header-test"); err == nil {
		t.Fatal("expected incomplete header")
	}
	if !strings.Contains(out.String(), "final:") {
		t.Fatal("missing final report")
	}
}
