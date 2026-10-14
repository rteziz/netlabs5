package main

import (
	"errors"
	"filetransfer/protocol"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
	return clientName, nil
}

func createUniqueFile(name string) (*os.File, string, error) {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)

	for i := 0; i < maxNameAttempts; i++ {
		candidate := name
		if i > 0 {
			candidate = fmt.Sprintf("%s_%d%s", stem, i, ext)
		}
		fullPath := filepath.Join(UploadsDir, candidate)

		file, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, uploadedFilePerm)
		if err == nil {
			return file, fullPath, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, "", err
		}
	}
	return nil, "", fmt.Errorf("no free name for %q", name)
}
