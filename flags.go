package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"time"
)

type Config struct {
	Group        string
	Port         int
	Iface        string
	SendInterval time.Duration
	PeerTimeout  time.Duration
	IsIPv6       bool
}

func parseFlags() (*Config, error) {
	group := flag.String(
		"group",
		"",
		"multicast-адрес группы (обязательный параметр), напр. 224.0.0.1 для IPv4 или ff02::1 для IPv6",
	)
	port := flag.Int(
		"port",
		9999,
		"UDP-порт, используемый для multicast-обмена",
	)
	iface := flag.String(
		"iface",
		"",
		"имя сетевого интерфейса для multicast (необязательно; если не задан — выбирается автоматически)",
	)
	interval := flag.Duration(
		"interval",
		2*time.Second,
		"интервал рассылки анонсов присутствия (\"я жив\")",
	)
	timeout := flag.Duration(
		"timeout",
		6*time.Second,
		"через сколько времени без анонсов от копии считать её исчезнувшей",
	)

	flag.Parse()

	if *group == "" {
		return nil, errors.New("не задан обязательный параметр -group (multicast-адрес группы)")
	}

	ip := net.ParseIP(*group)
	if ip == nil {
		return nil, fmt.Errorf("значение -group=%q не является корректным IP-адресом", *group)
	}
	if !ip.IsMulticast() {
		return nil, fmt.Errorf("адрес %q не является multicast-адресом (для IPv4 это диапазон 224.0.0.0-239.255.255.255, для IPv6 — адреса вида ffXX::)", *group)
	}

	isIPv6 := ip.To4() == nil

	if *timeout <= *interval {
		return nil, fmt.Errorf("-timeout (%s) должен быть больше, чем -interval (%s), иначе живые копии будут ошибочно считаться исчезнувшими", *timeout, *interval)
	}

	return &Config{
		Group:        *group,
		Port:         *port,
		Iface:        *iface,
		SendInterval: *interval,
		PeerTimeout:  *timeout,
		IsIPv6:       isIPv6,
	}, nil
}
