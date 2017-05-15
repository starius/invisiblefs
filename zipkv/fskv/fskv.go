package fskv

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type FsKV struct {
	root string
}

func New(root string) (*FsKV, error) {
	return &FsKV{
		root: root,
	}, nil
}

func (f *FsKV) Has(key string) (bool, error) {
	path := filepath.Join(f.root, key)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, fmt.Errorf("os.Stat(%q): %s", path, err)
	}
}

func (f *FsKV) Get(key string) ([]byte, error) {
	path := filepath.Join(f.root, key)
	return ioutil.ReadFile(path)
}

func (f *FsKV) GetAt(key string, offset, size int) ([]byte, error) {
	path := filepath.Join(f.root, key)
	bf, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("os.Open(%q): %s", path, err)
	}
	defer bf.Close()
	buf := make([]byte, size)
	if _, err := bf.ReadAt(buf, int64(offset)); err != nil {
		return nil, fmt.Errorf(
			"bf.ReadAt(%q, offset=%d, size=%d): %s",
			path, offset, size, err,
		)
	}
	return buf, nil
}

func (f *FsKV) Put(key string, value []byte) error {
	path := filepath.Join(f.root, key)
	return ioutil.WriteFile(path, value, 0600)
}

func (f *FsKV) Delete(key string) error {
	path := filepath.Join(f.root, key)
	return os.Remove(path)
}

func (f *FsKV) Sync() error {
	return nil
}
