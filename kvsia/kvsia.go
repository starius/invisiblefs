package kvsia

import (
	"fmt"
	"strings"

	"github.com/starius/invisiblefs/siaform/files"
)

type KvSia struct {
	f *files.Files
}

func New(f *files.Files) (*KvSia, error) {
	return &KvSia{f}, nil
}

func (k *KvSia) Has(key string) (bool, []byte, error) {
	has, err := k.f.Has("metadata-" + key)
	if err != nil {
		return false, nil, fmt.Errorf("f.Has: %v", err)
	}
	var metadata []byte
	if has {
		metadata, err = k.f.Get("metadata-" + key)
		if err != nil {
			return false, nil, fmt.Errorf("Failed to get metadata: %v", err)
		}
	}
	return has, metadata, nil
}

func (k *KvSia) Get(key string) ([]byte, []byte, error) {
	metadata, err := k.f.Get("metadata-" + key)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get metadata: %v", err)
	}
	data, err := k.f.Get("data-" + key)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get data: %v", err)
	}
	return data, metadata, nil
}

func (k *KvSia) GetAt(key string, offset, size int) ([]byte, []byte, error) {
	metadata, err := k.f.Get("metadata-" + key)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get metadata: %v", err)
	}
	data, err := k.f.GetAt("data-"+key, offset, size)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get data: %v", err)
	}
	return data, metadata, nil
}

func (k *KvSia) List() (map[string]int, error) {
	l, err := k.f.List()
	if err != nil {
		return nil, err
	}
	m := make(map[string]int)
	for key, size := range l {
		if strings.HasPrefix(key, "data-") {
			m[strings.TrimPrefix(key, "data-")] = size
		}
	}
	return m, nil
}

func (k *KvSia) Put(key string, value, metadata []byte) error {
	if err := k.f.Put("metadata-"+key, metadata); err != nil {
		return fmt.Errorf("Failed to put metadata: %v", err)
	}
	if err := k.f.Put("data-"+key, value); err != nil {
		return fmt.Errorf("Failed to put data: %v", err)
	}
	return nil
}

func (k *KvSia) Link(dstKey, srcKey string, metadata []byte) error {
	if err := k.f.Put("metadata-"+dstKey, metadata); err != nil {
		return fmt.Errorf("Failed to put metadata: %v", err)
	}
	if err := k.f.Link("data-"+dstKey, "data-"+srcKey); err != nil {
		return fmt.Errorf("Failed to link data: %v", err)
	}
	return nil
}

func (k *KvSia) Delete(key string) (metadata []byte, err error) {
	metadata, err = k.f.Get("metadata-" + key)
	if err != nil {
		return nil, fmt.Errorf("Failed to get metadata: %v", err)
	}
	if err := k.f.Delete("metadata-" + key); err != nil {
		return nil, fmt.Errorf("Failed to delete metadata: %v", err)
	}
	if err := k.f.Delete("data-" + key); err != nil {
		return nil, fmt.Errorf("Failed to delete data: %v", err)
	}
	return metadata, nil
}

func (k *KvSia) Sync() error {
	// TODO upload is_progress, upload pending
	return nil
}
