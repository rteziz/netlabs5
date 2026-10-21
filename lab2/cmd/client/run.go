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

func run() {
	if len(os.Args) != argCount {
		log.Fatalf("usage: %s <file> <host> <port>", os.Args[0])
	}
	if err := protocol.ValidatePort(os.Args[portArg]); err != nil {
		log.Fatal(err)
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
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

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

	conn, err := net.DialTimeout("tcp", address, protocol.ConnectTimeout)
	if err != nil {
		return err
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
		}
	}(conn)

	conn = &protocol.IdleConn{Conn: conn, Timeout: protocol.IdleTimeout}
	if err := protocol.WriteHeader(conn, header); err != nil {
		return err
	}
	if _, err := io.CopyN(conn, file, int64(header.Size)); err != nil {
		return err
	}
	current, err := file.Stat()
	if err != nil {
		return err
	}
	if current.Size() != info.Size() {
		return fmt.Errorf("file size changed during transfer: was %d, now %d", info.Size(), current.Size())
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
