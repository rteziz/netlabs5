package main

import (
	"errors"
	"filetransfer/protocol"
	"log"
	"net"
	"os"
	"time"
)

const (
	maxClients     = 16
	uploadsDirPerm = 0o755
)

func run() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <port>", os.Args[0])
	}
	port := os.Args[1]
	if err := protocol.ValidatePort(port); err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(UploadsDir, uploadsDirPerm); err != nil {
		log.Fatalf("cannot create uploads dir: %v", err)
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
		}
	}(listener)

	log.Printf("listening on %s", listener.Addr())

	slots := make(chan int, maxClients)
	for id := 1; id <= maxClients; id++ {
		slots <- id
	}
	var retryDelay time.Duration
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if temporary, ok := err.(net.Error); !ok || !temporary.Temporary() {
				log.Printf("accept stopped: %v", err)
				return
			}
			if retryDelay == 0 {
				retryDelay = 5 * time.Millisecond
			} else {
				retryDelay *= 2
			}
			if retryDelay > time.Second {
				retryDelay = time.Second
			}
			log.Printf("accept: %v; retry in %s", err, retryDelay)
			time.Sleep(retryDelay)
			continue
		}
		retryDelay = 0
		select {
		case id := <-slots:
			go func(conn net.Conn, id int) {
				defer func() { slots <- id }()
				handleClient(conn, id)
			}(conn, id)
		default:
			log.Printf("[%s] rejected: client limit (%d) reached", conn.RemoteAddr(), maxClients)
			rejectClient(conn)
		}
	}
}

func rejectClient(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("[%s] cannot close rejected connection: %v", conn.RemoteAddr(), err)
		}
	}()
	if err := conn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		log.Printf("[%s] cannot set rejection deadline: %v", conn.RemoteAddr(), err)
		return
	}
	if _, err := conn.Write([]byte{protocol.StatusBusy}); err != nil {
		log.Printf("[%s] cannot send busy status: %v", conn.RemoteAddr(), err)
	}
}
