package main

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"
)

const ReportInterval = 3 * time.Second

type Meter struct {
	total atomic.Int64
	start time.Time
}

func (m *Meter) Write(p []byte) (int, error) {
	m.total.Add(int64(len(p)))
	return len(p), nil
}

func (m *Meter) Report(client string, stop <-chan struct{}) {
	start := m.start
	if start.IsZero() {
		start = time.Now()
	}
	lastTime := start
	var lastTotal int64

	printRates := func(final bool) {
		now := time.Now()
		total := m.total.Load()
		kind := "periodic"
		if final {
			kind = "final"
		}
		log.Printf("[%s] %s: instant: %s, average: %s",
			client, kind,
			formatRate(bytesPerSecond(total-lastTotal, now.Sub(lastTime))),
			formatRate(bytesPerSecond(total, now.Sub(start))))
		lastTime, lastTotal = now, total
	}

	ticker := time.NewTicker(ReportInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			printRates(false)
		case <-stop:
			printRates(true)
			return
		}
	}
}

func bytesPerSecond(bytes int64, elapsed time.Duration) float64 {
	if elapsed <= 0 {
		return 0
	}
	return float64(bytes) / elapsed.Seconds()
}

func formatRate(rate float64) string {
	switch {
	case rate >= 1e9:
		return fmt.Sprintf("%.2f GB/s", rate/1e9)
	case rate >= 1e6:
		return fmt.Sprintf("%.2f MB/s", rate/1e6)
	case rate >= 1e3:
		return fmt.Sprintf("%.2f KB/s", rate/1e3)
	default:
		return fmt.Sprintf("%.2f B/s", rate)
	}
}

func startMeter(client string) (*Meter, func()) {
	m := &Meter{start: time.Now()}
	stop, done := make(chan struct{}), make(chan struct{})
	go func() { defer close(done); m.Report(client, stop) }()
	return m, func() { close(stop); <-done }
}
