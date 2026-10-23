package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackmesh/go-media-gateway/internal/domain"
)

func TestDiskBlobHashesAndRejects(t *testing.T) {
	store, err := NewDiskBlob(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("media")
	n, hash, err := store.Put(context.Background(), "demo", bytes.NewReader(body), 16, "")
	if err != nil || n != int64(len(body)) {
		t.Fatalf("put %d %v", n, err)
	}
	sum := sha256.Sum256(body)
	if hash != hex.EncodeToString(sum[:]) {
		t.Fatalf("hash %s", hash)
	}
	if _, err := os.Stat(filepath.Join(store.root, "demo", hash)); err != nil {
		t.Fatal(err)
	}

	_, _, err = store.Put(context.Background(), "demo", bytes.NewReader([]byte("123456")), 4, "")
	if err != domain.ErrTooLarge {
		t.Fatalf("size: %v", err)
	}
	_, _, err = store.Put(context.Background(), "demo", bytes.NewReader(body), 16, hex.EncodeToString(bytes.Repeat([]byte{0}, 32)))
	if err != domain.ErrHashMismatch {
		t.Fatalf("mismatch: %v", err)
	}
	_, _, err = store.Put(context.Background(), "../demo", bytes.NewReader(body), 16, "")
	if err != domain.ErrTenant {
		t.Fatalf("tenant: %v", err)
	}
}
