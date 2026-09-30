package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func startSender(cfg *Config, selfID string, iface *net.Interface, stop <-chan struct{}) error {
	network := networkKind(cfg.IsIPv6)

	groupAddr, err := net.ResolveUDPAddr(network, net.JoinHostPort(cfg.Group, strconv.Itoa(cfg.Port)))
	if err != nil {
		return fmt.Errorf("не удалось разобрать адрес multicast-группы: %w", err)
	}

	if cfg.IsIPv6 && groupAddr.IP.IsLinkLocalMulticast() {
		groupAddr.Zone = iface.Name
	}

	conn, err := net.DialUDP(network, nil, groupAddr)
	if err != nil {
		return fmt.Errorf("не удалось открыть сокет для отправки анонсов: %w", err)
	}

	if err := setMulticastInterface(conn, iface, cfg.IsIPv6); err != nil {
		log.Printf("предупреждение: не удалось явно закрепить исходящий multicast-интерфейс %s: %v (будет использован интерфейс по умолчанию ОС)", iface.Name, err)
	}

	go sendLoop(conn, selfID, cfg.SendInterval, stop)

	log.Printf("отправляю анонсы присутствия в группу %s каждые %s", groupAddr, cfg.SendInterval)
	return nil
}

func setMulticastInterface(conn *net.UDPConn, iface *net.Interface, isIPv6 bool) error {
	if isIPv6 {
		return ipv6.NewPacketConn(conn).SetMulticastInterface(iface)
	}
	return ipv4.NewPacketConn(conn).SetMulticastInterface(iface)
}

func sendLoop(conn *net.UDPConn, selfID string, interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer conn.Close()

	sendAnnounce(conn, selfID)

	for {
		select {
		case <-ticker.C:
			sendAnnounce(conn, selfID)
		case <-stop:
			return
		}
	}
}

func sendAnnounce(conn *net.UDPConn, selfID string) {
	if _, err := conn.Write([]byte(selfID)); err != nil {
		log.Printf("ошибка при отправке анонса присутствия: %v", err)
	}
}
