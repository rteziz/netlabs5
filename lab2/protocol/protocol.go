package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
	"unicode/utf8"
)

const (
	StatusFail  byte = 0
	StatusOK    byte = 1
	StatusBusy  byte = 2
	MaxNameLen       = 4096
	MaxFileSize      = 1 << 40
)

var byteOrder = binary.BigEndian

type Header struct {
	Name string
	Size uint64
}

func WriteHeader(w io.Writer, h Header) error {
	if err := h.Validate(); err != nil {
		return err
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
	if nameLen == 0 || nameLen > MaxNameLen {
		return Header{}, fmt.Errorf("invalid name length %d (maximum %d)", nameLen, MaxNameLen)
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
	h := Header{Name: string(name), Size: size}
	if err := h.Validate(); err != nil {
		return Header{}, err
	}
	return h, nil
}

func (h Header) Validate() error {
	if err := ValidateFileName(h.Name); err != nil {
		return err
	}
	if h.Size > MaxFileSize {
		return fmt.Errorf("file size %d exceeds %d bytes", h.Size, uint64(MaxFileSize))
	}
	return nil
}

func ValidateFileName(name string) error {
	if len(name) == 0 || len(name) > MaxNameLen {
		return fmt.Errorf("file name must contain 1 to %d bytes", MaxNameLen)
	}
	if !utf8.ValidString(name) {
		return fmt.Errorf("file name is not valid UTF-8")
	}
	return nil
}
