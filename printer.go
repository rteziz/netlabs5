package main

import (
	"fmt"
	"log"
)

func printPeers(tracker *PeerTracker, reason string) {
	addrs := tracker.Snapshot()

	log.Println(reason)

	if len(addrs) == 0 {
		fmt.Println("  других живых копий приложения в сети не обнаружено")
		return
	}

	fmt.Println("  текущий список живых копий приложения:")
	for _, addr := range addrs {
		fmt.Printf("    - %s\n", addr)
	}
}
