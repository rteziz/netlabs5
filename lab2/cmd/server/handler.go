package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"filetransfer/protocol"
)

func handleClient(conn net.Conn, id int) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {

		}
	}(conn)
	conn = &protocol.IdleConn{Conn: conn, Timeout: protocol.IdleTimeout}
	client := fmt.Sprintf("client-%d %s", id, conn.RemoteAddr())
	log.Printf("[%s] connected", client)

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
		if removeErr := os.Remove(path); removeErr != nil {
			return errors.Join(err, fmt.Errorf("remove partial file: %w", removeErr))
		}
	}
	return err
}

func copyBody(conn net.Conn, dst io.Writer, size uint64, client string) (int64, error) {
	meter := &Meter{}
	stop := make(chan struct{})
	done := make(chan struct{})
	defer func() {
		close(stop)
		<-done
	}()
	go func() {
		defer close(done)
		meter.Report(client, stop)
	}()

	body := io.TeeReader(io.LimitReader(conn, int64(size)), meter)
	return io.Copy(dst, body)
}
