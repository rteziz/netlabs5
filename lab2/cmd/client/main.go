package main

import (
	"filetransfer/protocol"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
)

const (
	argCount = 4
	pathArg  = 1
	hostArg  = 2
	portArg  = 3
)

func main() {
	if len(os.Args) != argCount {
		log.Fatalf("usage: %s <file> <host> <port>", os.Args[0])
	}
	address := net.JoinHostPort(os.Args[hostArg], os.Args[portArg])

	if err := sendFile(os.Args[pathArg], address); err != nil {
		fmt.Println("Transfer FAILED:", err)
		os.Exit(1)
	}
	fmt.Println("Transfer successful")
}

func sendFile(path, address string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", path)
	}

	header := protocol.Header{Name: filepath.Base(path), Size: uint64(info.Size())}
	if err := header.Validate(); err != nil {
		return err
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := protocol.WriteHeader(conn, header); err != nil {
		return err
	}
	if _, err := io.Copy(conn, file); err != nil {
		return err
	}
	return waitForStatus(conn)
}

func waitForStatus(conn net.Conn) error {
	status := make([]byte, 1)
	if _, err := io.ReadFull(conn, status); err != nil {
		return fmt.Errorf("no answer from server: %w", err)
	}
	if status[0] != protocol.StatusOK {
		return fmt.Errorf("server reported a failure")
	}
	return nil
}
