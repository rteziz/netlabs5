package main

import (
	"log"
	"sort"
	"sync"
	"time"
)

const maxPeers = 1000

type peerInfo struct {
	addr     string
	lastSeen time.Time
}

type PeerTracker struct {
	mu         sync.Mutex
	peers      map[string]peerInfo
	atCapacity bool
}

func newPeerTracker() *PeerTracker {
	return &PeerTracker{
		peers: make(map[string]peerInfo),
	}
}

func (t *PeerTracker) Touch(id, addr string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	_, existed := t.peers[id]
	if !existed {
		if len(t.peers) >= maxPeers {
			if !t.atCapacity {
				log.Printf("предупреждение: достигнут предел в %d отслеживаемых копий, новые ID игнорируются до освобождения места", maxPeers)
				t.atCapacity = true
			}
			return false
		}
		t.atCapacity = false
	}

	t.peers[id] = peerInfo{
		addr:     addr,
		lastSeen: time.Now(),
	}
	return !existed
}

func (t *PeerTracker) RemoveExpired(timeout time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	changed := false
	now := time.Now()
	for id, info := range t.peers {
		if now.Sub(info.lastSeen) > timeout {
			delete(t.peers, id)
			changed = true
		}
	}
	return changed
}

func (t *PeerTracker) Snapshot() []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	addrs := make([]string, 0, len(t.peers))
	for _, info := range t.peers {
		addrs = append(addrs, info.addr)
	}
	sort.Strings(addrs)
	return addrs
}
