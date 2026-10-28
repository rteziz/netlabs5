package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"filetransfer/protocol"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"
)

func run() {
	if len(os.Args) != 4 {
		log.Fatalf("usage: %s <file> <host> <port>", os.Args[0])
	}
	path, host, port := os.Args[1], os.Args[2], os.Args[3]
	if err := protocol.ValidatePort(port); err != nil {
		log.Fatal(err)
	}
	if err := sendFile(path, net.JoinHostPort(host, port)); err != nil {
		fmt.Println("Transfer FAILED:", err)
		os.Exit(1)
	}
	fmt.Printf("Transfer successful: %q\n", filepath.Base(path))
	fmt.Println("Local contents check: SHA-256 matches")
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

	var before []byte
	before, err = hashFile(file)
	if err != nil {
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
		return sendError(conn, err)
	}
	sentHash := sha256.New()
	destination := io.MultiWriter(conn, sentHash)
	if _, err := io.CopyN(destination, file, int64(header.Size)); err != nil {
		return sendError(conn, err)
	}
	current, err := file.Stat()
	if err != nil {
		return err
	}
	if current.Size() != info.Size() {
		return fmt.Errorf("file size changed during transfer")
	}
	if !bytes.Equal(before, sentHash.Sum(nil)) {
		return fmt.Errorf("file contents changed during transfer")
	}
	after, err := hashFile(file)
	if err != nil {
		return err
	}
	if !bytes.Equal(before, after) {
		return fmt.Errorf("file contents changed during transfer")
	}
	return readStatus(conn)
}

func readStatus(conn net.Conn) error {
	var status [1]byte
	if _, err := io.ReadFull(conn, status[:]); err != nil {
		return fmt.Errorf("cannot read server response: %w", err)
	}
	switch status[0] {
	case protocol.StatusOK:
		return nil
	case protocol.StatusFail:
		return fmt.Errorf("server could not receive or save the file")
	case protocol.StatusBusy:
		return fmt.Errorf("server is busy: no free slots; try again later")
	default:
		return fmt.Errorf("unknown server status: %d", status[0])
	}
}

func sendError(conn net.Conn, sendErr error) error {
	// Bypass IdleConn so it cannot overwrite this shorter response deadline.
	if wrapped, ok := conn.(*protocol.IdleConn); ok {
		conn = wrapped.Conn
	}
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.CloseWrite()
	}
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return errors.Join(sendErr, err)
	}
	return errors.Join(sendErr, readStatus(conn))
}
