package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
)

func startListener(cfg *Config, selfID string, iface *net.Interface, tracker *PeerTracker) error {
	network := networkKind(cfg.IsIPv6)

	groupAddr, err := net.ResolveUDPAddr(network, net.JoinHostPort(cfg.Group, strconv.Itoa(cfg.Port)))
	if err != nil {
		return fmt.Errorf("не удалось разобрать адрес multicast-группы: %w", err)
	}

	conn, err := net.ListenMulticastUDP(network, iface, groupAddr)
	if err != nil {
		return fmt.Errorf("не удалось присоединиться к multicast-группе: %w", err)
	}

	go receiveLoop(conn, selfID, tracker)

	log.Printf("слушаю multicast-группу %s на интерфейсе %s", groupAddr, iface.Name)
	return nil
}

func receiveLoop(conn *net.UDPConn, selfID string, tracker *PeerTracker) {
	buf := make([]byte, 1024)

	for {
		n, srcAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("приём multicast-сообщений остановлен: %v", err)
			return
		}

		receivedID := string(buf[:n])

		if receivedID == selfID {
			continue
		}

		peerIP := srcAddr.IP.String()

		if tracker.Touch(receivedID, peerIP) {
			printPeers(tracker, fmt.Sprintf("обнаружена новая копия приложения: %s", peerIP))
		}
	}
}
