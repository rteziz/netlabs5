package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"filetransfer/protocol"
)

func handleClient(conn net.Conn) {
	defer conn.Close()
	client := conn.RemoteAddr().String()

	status := protocol.StatusOK
	if err := receiveFile(conn, client); err != nil {
		log.Printf("[%s] failed: %v", client, err)
		status = protocol.StatusFail
	} else {
		log.Printf("[%s] file received successfully", client)
	}

	if _, err := conn.Write([]byte{status}); err != nil {
		log.Printf("[%s] cannot send status: %v", client, err)
	}
}

func receiveFile(conn net.Conn, client string) error {
	header, err := protocol.ReadHeader(conn)
	if err != nil {
		return err
	}
	name, err := safeFileName(header.Name)
	if err != nil {
		return err
	}
	file, path, err := createUniqueFile(name)
	if err != nil {
		return err
	}

	received, err := copyBody(conn, file, header.Size, client)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err == nil && received != int64(header.Size) {
		err = fmt.Errorf("size mismatch: expected %d bytes, got %d", header.Size, received)
	}
	if err != nil {
		os.Remove(path)
	}
	return err
}

func copyBody(conn net.Conn, dst io.Writer, size uint64, client string) (int64, error) {
	meter := &Meter{}
	stop := make(chan struct{})
	defer close(stop)
	go meter.Report(client, stop)

	body := io.TeeReader(io.LimitReader(conn, int64(size)), meter)
	return io.Copy(dst, body)
}
