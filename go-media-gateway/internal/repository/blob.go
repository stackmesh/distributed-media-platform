package repository

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/stackmesh/go-media-gateway/internal/domain"
)

// DiskBlob stores object bytes by tenant and sha256. The file is fsync'd
// before the gateway records metadata.
type DiskBlob struct {
	root string
}

func NewDiskBlob(root string) (*DiskBlob, error) {
	if root == "" {
		return nil, errors.New("blob directory is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, err
	}
	return &DiskBlob{root: abs}, nil
}

func (d *DiskBlob) Put(ctx context.Context, tenant string, body io.Reader, max int64, expectHash string) (int64, string, error) {
	if err := ctx.Err(); err != nil {
		return 0, "", err
	}
	if max <= 0 {
		return 0, "", domain.ErrTooLarge
	}
	if !domain.ValidTenant(tenant) {
		return 0, "", domain.ErrTenant
	}
	expectHash = strings.ToLower(strings.TrimSpace(expectHash))
	if expectHash != "" && !domain.ValidHash(expectHash) {
		return 0, "", domain.ErrHash
	}

	dir := filepath.Join(d.root, tenant)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return 0, "", err
	}
	tmp, err := os.CreateTemp(dir, ".part-*")
	if err != nil {
		return 0, "", err
	}
	tmpName := tmp.Name()
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	hasher := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, hasher), io.LimitReader(body, max+1))
	if err != nil {
		_ = tmp.Close()
		return 0, "", err
	}
	if n == 0 {
		_ = tmp.Close()
		return 0, "", domain.ErrEmpty
	}
	if n > max {
		_ = tmp.Close()
		return 0, "", domain.ErrTooLarge
	}
	sum := hex.EncodeToString(hasher.Sum(nil))
	if expectHash != "" && subtle.ConstantTimeCompare([]byte(sum), []byte(expectHash)) != 1 {
		_ = tmp.Close()
		return 0, "", domain.ErrHashMismatch
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return 0, "", err
	}
	if err := tmp.Close(); err != nil {
		return 0, "", err
	}

	final, err := d.path(tenant, sum)
	if err != nil {
		return 0, "", err
	}
	if err := os.Rename(tmpName, final); err != nil {
		return 0, "", err
	}
	success = true
	return n, sum, nil
}

func (d *DiskBlob) Remove(ctx context.Context, tenant, hash string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := d.path(tenant, hash)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (d *DiskBlob) path(tenant, hash string) (string, error) {
	if !domain.ValidTenant(tenant) {
		return "", domain.ErrTenant
	}
	if !domain.ValidHash(hash) {
		return "", domain.ErrHash
	}
	full := filepath.Join(d.root, tenant, hash)
	rel, err := filepath.Rel(d.root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", domain.ErrTenant
	}
	return full, nil
}
