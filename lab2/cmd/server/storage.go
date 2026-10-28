package main

import (
	"crypto/sha256"
	"errors"
	"filetransfer/protocol"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	UploadsDir       = "uploads"
	uploadedFilePerm = 0o644
	maxNameAttempts  = 10000
)

func safeFileName(clientName string) (string, error) {
	if err := protocol.ValidateFileName(clientName); err != nil {
		return "", err
	}
	// Map names that cannot be used safely as one local path component.
	if clientName == "." || clientName == ".." || strings.ContainsAny(clientName, "/\\\x00") {
		digest := sha256.Sum256([]byte(clientName))
		return fmt.Sprintf("file-%x", digest[:16]), nil
	}
	return clientName, nil
}

func createUniqueFile(name string) (*os.File, string, error) {
	compact := false
	digest := sha256.Sum256([]byte(name))
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)

	for i := 0; i < maxNameAttempts; i++ {
		candidate := name
		if i > 0 {
			candidate = fmt.Sprintf("%s_%d%s", stem, i, ext)
		}
		if compact {
			candidate = fmt.Sprintf("file-%x_%d", digest[:16], i)
		}
		fullPath := filepath.Join(UploadsDir, candidate)

		file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, uploadedFilePerm)
		if err == nil {
			return file, fullPath, nil
		}
		if errors.Is(err, syscall.ENAMETOOLONG) && !compact {
			compact = true
			i--
			continue
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, "", err
		}
	}
	return nil, "", fmt.Errorf("no free name for %q", name)
}
