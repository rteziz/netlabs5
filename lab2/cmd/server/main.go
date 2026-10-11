package main

import (
	"log"
	"net"
	"os"
)

const (
	argCount       = 2
	portArg        = 1
	uploadsDirPerm = 0o755 //755 rwxr-xr-x
)

func main() {
	if len(os.Args) != argCount {
		log.Fatalf("usage: %s <port>", os.Args[0])
	}
	if err := os.MkdirAll(UploadsDir, uploadsDirPerm); err != nil {
		log.Fatalf("cannot create uploads dir: %v", err)
	}

	listener, err := net.Listen("tcp", ":"+os.Args[portArg])
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	log.Printf("listening on %s", listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handleClient(conn)
	}
}
