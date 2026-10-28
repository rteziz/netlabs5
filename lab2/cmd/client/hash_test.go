package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestHashDetectsSameSizeChangeAndRewinds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data")
	if err := os.WriteFile(path, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	before, err := hashFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("xyz"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := hashFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before, after) {
		t.Fatal("same-size change missed")
	}
	buf := make([]byte, 3)
	if _, err := f.Read(buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "xyz" {
		t.Fatal("file not rewound")
	}
}
