package main

import (
	"bytes"
	"crypto/sha256"
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
	fmt.Printf("Transfer successful: %q\n", filepath.Base(os.Args[pathArg]))
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

	before, err := hashFile(file)
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
	if err := waitForAdmission(conn); err != nil {
		return err
	}
	if err := protocol.WriteHeader(conn, header); err != nil {
		return err
	}
	sentHash := sha256.New()
	if _, err := io.CopyN(io.MultiWriter(conn, sentHash), file, int64(header.Size)); err != nil {
		return err
	}
	current, err := file.Stat()
	if err != nil {
		return err
	}
	if current.Size() != info.Size() {
		return fmt.Errorf("file size changed during transfer: was %d, now %d", info.Size(), current.Size())
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
	return waitForStatus(conn)
}

func waitForStatus(conn net.Conn) error {
	status := make([]byte, 1)
	if _, err := io.ReadFull(conn, status); err != nil {
		return fmt.Errorf("no answer from server: %w", err)
	}
	switch status[0] {
	case protocol.StatusOK:
		return nil
	case protocol.StatusFail:
		return fmt.Errorf("server reported a failure while receiving the file (status %d); check server logs", status[0])
	default:
		return fmt.Errorf("unexpected final status %d; rebuild both client and server from the same sources", status[0])
	}
}

func waitForAdmission(conn net.Conn) error {
	var status [1]byte
	if _, err := io.ReadFull(conn, status[:]); err != nil {
		return fmt.Errorf("no admission response from server: %w", err)
	}
	switch status[0] {
	case protocol.StatusReady:
		return nil
	case protocol.StatusBusy:
		return fmt.Errorf("server is full: no free slots for receiving files; try again later")
	default:
		return fmt.Errorf("unexpected admission status: %d", status[0])
	}
}

func hashFile(file *os.File) ([]byte, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
