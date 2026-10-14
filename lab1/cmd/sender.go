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

const multicastScopeTTL = 1

func startSender(cfg *Config, selfID, hostname string, iface *net.Interface, stop <-chan struct{}) error {
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

	if err := setMulticastInterface(conn, iface, cfg.IsIPv6); err != nil { //вот
		conn.Close()
		return fmt.Errorf("не удалось закрепить исходящий multicast-интерфейс %s: %w", iface.Name, err)
	}

	if err := setMulticastScope(conn, cfg.IsIPv6); err != nil {
		conn.Close()
		return fmt.Errorf("не удалось ограничить TTL/hop limit multicast-пакетов: %w", err)
	}

	go sendLoop(conn, buildHeartbeat(selfID, hostname), cfg.SendInterval, stop)

	log.Printf("отправляю анонсы присутствия в группу %s каждые %s", groupAddr, cfg.SendInterval)
	return nil
}

func setMulticastInterface(conn *net.UDPConn, iface *net.Interface, isIPv6 bool) error {
	if isIPv6 {
		return ipv6.NewPacketConn(conn).SetMulticastInterface(iface)
	}
	return ipv4.NewPacketConn(conn).SetMulticastInterface(iface)
}

func setMulticastScope(conn *net.UDPConn, isIPv6 bool) error {
	if isIPv6 {
		return ipv6.NewPacketConn(conn).SetMulticastHopLimit(multicastScopeTTL)
	}
	return ipv4.NewPacketConn(conn).SetMulticastTTL(multicastScopeTTL)
}

func sendLoop(conn *net.UDPConn, payload []byte, interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer conn.Close()

	sendAnnounce(conn, payload)

	for {
		select {
		case <-ticker.C:
			sendAnnounce(conn, payload)
		case <-stop:
			return
		}
	}
}

func sendAnnounce(conn *net.UDPConn, payload []byte) {
	if _, err := conn.Write(payload); err != nil {
		log.Printf("ошибка при отправке анонса присутствия: %v", err)
	}
}
