package main

import (
	"filetransfer/protocol"
	"log"
	"net"
	"os"
)

const (
	maxClients     = 16
	uploadsDirPerm = 0o755 //755 rwxr-xr-x
)

func main() {
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
	defer listener.Close()
	log.Printf("listening on %s", listener.Addr())

	slots := make(chan int, maxClients)
	for id := 1; id <= maxClients; id++ {
		slots <- id
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		select {
		case id := <-slots:
			go func(conn net.Conn, id int) {
				defer func() { slots <- id }()
				handleClient(conn, id)
			}(conn, id)
		default:
			log.Printf("[%s] rejected: client limit (%d) reached", conn.RemoteAddr(), maxClients)
			conn.Close()
		}
	}
}
