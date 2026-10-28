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
		status = protocol.StatusFail
		log.Printf("[%s] failed: %v", client, err)
	}

	if _, err := conn.Write([]byte{status}); err != nil {
		log.Printf("[%s] cannot send status: %v", client, err)
	}
}

func receiveFile(conn net.Conn, client string) error {
	meter, finish := startMeter(client)
	defer finish()
	source := io.TeeReader(conn, meter)
	header, err := protocol.ReadHeader(source)
	if err != nil {
		return err
	}
	meter.fileName.Store(header.Name)
	name, err := safeFileName(header.Name)
	if err != nil {
		return err
	}
	file, path, err := createUniqueFile(name)
	if err != nil {
		return err
	}

	received, err := io.Copy(file, io.LimitReader(source, int64(header.Size)))
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err == nil && received != int64(header.Size) {
		err = fmt.Errorf("size mismatch: expected %d bytes, got %d", header.Size, received)
	}
	if err == nil {
		saved, statErr := os.Stat(path)
		if statErr != nil {
			err = fmt.Errorf("stat saved file: %w", statErr)
		} else if saved.Size() != int64(header.Size) {
			err = fmt.Errorf("disk size mismatch: expected %d bytes, got %d", header.Size, saved.Size())
		}
	}
	if err != nil {
		if removeErr := os.Remove(path); removeErr != nil {
			return errors.Join(err, fmt.Errorf("remove partial file: %w", removeErr))
		}
	}
	if err == nil {
		log.Printf("[%s] file %q received successfully; saved as %q", client, header.Name, path)
	}
	return err
}
