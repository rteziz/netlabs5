package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := parseFlags()
	if err != nil {
		log.Fatalf("ошибка конфигурации: %v", err)
	}

	selfID, err := generateInstanceID()
	if err != nil {
		log.Fatal(err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("не удалось получить hostname: %v", err)
	}
	if !isValidHostname(hostname) {
		log.Fatalf("некорректный hostname: нужны ASCII-буквы, цифры, дефисы и точки; не более %d байт, метки до 63 байт", maxHostnameLen)
	}
	log.Printf("идентификатор этого экземпляра приложения: %s", selfID)

	protoName := "IPv4"
	if cfg.IsIPv6 {
		protoName = "IPv6"
	}
	log.Printf("используется протокол %s (определено автоматически по адресу группы %s)", protoName, cfg.Group)

	tracker := newPeerTracker()

	iface, err := resolveInterface(cfg.Iface, cfg.IsIPv6)
	if err != nil {
		log.Fatalf("не удалось определить сетевой интерфейс: %v", err)
	}
	log.Printf("используется сетевой интерфейс: %s", iface.Name)

	stop := make(chan struct{})

	if err := startListener(cfg, selfID, iface, tracker); err != nil {
		log.Fatalf("не удалось запустить приём multicast-сообщений: %v", err)
	}

	if err := startSender(cfg, selfID, hostname, iface, stop); err != nil {
		log.Fatalf("не удалось запустить отправку multicast-сообщений: %v", err)
	}

	go cleanupLoop(cfg, tracker, stop)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("получен сигнал завершения, останавливаю приложение...")
	close(stop)
	time.Sleep(200 * time.Millisecond)
}

func cleanupLoop(cfg *Config, tracker *PeerTracker, stop <-chan struct{}) {
	ticker := time.NewTicker(cfg.SendInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if tracker.RemoveExpired(cfg.PeerTimeout) {
				printPeers(tracker, "одна или несколько копий приложения исчезли из сети")
			}
		case <-stop:
			return
		}
	}
}
