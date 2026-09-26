package infrastructure

import (
	"context"
	"os"
	"path/filepath"

	"traineebox/internal/curriculum/domain/errs"
)

type DiskBlobStore struct {
	root string
}

func NewDiskBlobStore(root string) *DiskBlobStore {
	return &DiskBlobStore{root: root}
}

func (s *DiskBlobStore) Put(_ context.Context, key string, data []byte) error {
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.path(key), data, 0o644)
}

func (s *DiskBlobStore) Get(_ context.Context, key string) ([]byte, error) {
	data, err := os.ReadFile(s.path(key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	return data, nil
}

func (s *DiskBlobStore) Delete(_ context.Context, key string) error {
	err := os.Remove(s.path(key))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *DiskBlobStore) path(key string) string {
	return filepath.Join(s.root, filepath.Base(key))
}
