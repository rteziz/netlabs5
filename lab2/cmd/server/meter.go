package main

import (
	"log"
	"sync/atomic"
	"time"
)

const ReportInterval = 3 * time.Second

type Meter struct {
	total atomic.Int64
}

func (m *Meter) Write(p []byte) (int, error) {
	m.total.Add(int64(len(p)))
	return len(p), nil
}

func (m *Meter) Report(client string, stop <-chan struct{}) {
	start := time.Now()
	lastTime := start
	var lastTotal int64
	printed := false

	printRates := func() {
		now := time.Now()
		total := m.total.Load()
		log.Printf("[%s] instant: %.0f B/s, average: %.0f B/s",
			client,
			bytesPerSecond(total-lastTotal, now.Sub(lastTime)),
			bytesPerSecond(total, now.Sub(start)))
		lastTime, lastTotal, printed = now, total, true
	}

	ticker := time.NewTicker(ReportInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			printRates()
		case <-stop:
			if !printed {
				printRates()
			}
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
