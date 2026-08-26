package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type ImageStore struct {
	Dir string
}

func (s ImageStore) Save(_ context.Context, name string, r io.Reader) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(s.Dir, name)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (s ImageStore) Path(name string) string {
	if name == "" {
		return ""
	}
	return filepath.Join(s.Dir, name)
}

func (s ImageStore) Remove(name string) error {
	if name == "" {
		return nil
	}
	err := os.Remove(filepath.Join(s.Dir, name))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s ImageStore) Exists(name string) bool {
	if name == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(s.Dir, name))
	return err == nil
}
