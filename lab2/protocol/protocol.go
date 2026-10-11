package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	MaxNameLen  = 4096
	MaxFileSize = 1 << 40

	StatusOK   byte = 0
	StatusFail byte = 1
)

var byteOrder = binary.BigEndian

type Header struct {
	Name string
	Size uint64
}

func WriteHeader(w io.Writer, h Header) error {
	if len(h.Name) > MaxNameLen {
		return fmt.Errorf("file name is longer than %d bytes", MaxNameLen)
	}
	if h.Size > MaxFileSize {
		return fmt.Errorf("file is larger than %d bytes", uint64(MaxFileSize))
	}
	if err := binary.Write(w, byteOrder, uint16(len(h.Name))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, h.Name); err != nil {
		return err
	}
	return binary.Write(w, byteOrder, h.Size)
}

func ReadHeader(r io.Reader) (Header, error) {
	var nameLen uint16
	if err := binary.Read(r, byteOrder, &nameLen); err != nil {
		return Header{}, fmt.Errorf("reading name length: %w", err)
	}
	if nameLen > MaxNameLen {
		return Header{}, fmt.Errorf("name length %d exceeds %d", nameLen, MaxNameLen)
	}

	name := make([]byte, nameLen)
	if _, err := io.ReadFull(r, name); err != nil {
		return Header{}, fmt.Errorf("reading name: %w", err)
	}
	if !utf8.Valid(name) {
		return Header{}, fmt.Errorf("file name is not valid UTF-8")
	}

	var size uint64
	if err := binary.Read(r, byteOrder, &size); err != nil {
		return Header{}, fmt.Errorf("reading file size: %w", err)
	}
	if size > MaxFileSize {
		return Header{}, fmt.Errorf("file size %d exceeds %d", size, uint64(MaxFileSize))
	}
	return Header{Name: string(name), Size: size}, nil
}
