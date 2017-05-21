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

func (f *FsKV) Has(key string) (bool, []byte, error) {
	path := filepath.Join(f.root, key)
	_, err := os.Stat(path)
	if err == nil {
		return true, nil, nil
	} else if os.IsNotExist(err) {
		return false, nil, nil
	} else {
		return false, nil, fmt.Errorf("os.Stat(%q): %s", path, err)
	}
}

func (f *FsKV) Get(key string) ([]byte, []byte, error) {
	path := filepath.Join(f.root, key)
	data, err := ioutil.ReadFile(path)
	return data, nil, err
}

func (f *FsKV) GetAt(key string, offset, size int) ([]byte, []byte, error) {
	path := filepath.Join(f.root, key)
	bf, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("os.Open(%q): %s", path, err)
	}
	defer bf.Close()
	buf := make([]byte, size)
	if _, err := bf.ReadAt(buf, int64(offset)); err != nil {
		return nil, nil, fmt.Errorf(
			"bf.ReadAt(%q, offset=%d, size=%d): %s",
			path, offset, size, err,
		)
	}
	return buf, nil, nil
}

func (f *FsKV) Put(key string, value, metadata []byte) error {
	if len(metadata) > 0 {
		return fmt.Errorf("fskv doesn't support metadata")
	}
	path := filepath.Join(f.root, key)
	return ioutil.WriteFile(path, value, 0600)
}

func (f *FsKV) Delete(key string) ([]byte, error) {
	path := filepath.Join(f.root, key)
	return nil, os.Remove(path)
}

func (f *FsKV) Sync() error {
	return nil
}
