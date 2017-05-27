package mem

import (
	"fmt"
	"sync"
)

type file struct {
	data, metadata []byte
}

type Mem struct {
	files map[string]file
	mu    sync.RWMutex
}

func New() (*Mem, error) {
	return &Mem{
		files: make(map[string]file),
	}, nil
}

func (m *Mem) Has(key string) (bool, []byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if f, has := m.files[key]; has {
		return true, f.metadata, nil
	} else {
		return false, nil, nil
	}
}

func (m *Mem) Get(key string) ([]byte, []byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if f, has := m.files[key]; has {
		return f.data, f.metadata, nil
	} else {
		return nil, nil, fmt.Errorf("no key %q", key)
	}
}

func (m *Mem) GetAt(key string, offset, size int) ([]byte, []byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if offset < 0 {
		return nil, nil, fmt.Errorf("offset=%d", offset)
	}
	if size < 0 {
		return nil, nil, fmt.Errorf("size=%d", size)
	}
	if f, has := m.files[key]; has {
		if offset > len(f.data) {
			return nil, f.metadata, nil
		}
		if offset+size > len(f.data) {
			size = len(f.data) - offset
		}
		return f.data[offset : offset+size], f.metadata, nil
	} else {
		return nil, nil, fmt.Errorf("no key %q", key)
	}
}

func (m *Mem) List() (map[string]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]int)
	for key, f := range m.files {
		result[key] = len(f.data)
	}
	return result, nil
}

func (m *Mem) Put(key string, value, metadata []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[key] = file{
		data:     value,
		metadata: metadata,
	}
	return nil
}

func (m *Mem) Link(dstKey, srcKey string, metadata []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, has := m.files[srcKey]
	if !has {
		return fmt.Errorf("no key %q", srcKey)
	}
	m.files[dstKey] = file{
		data:     f.data,
		metadata: metadata,
	}
	return nil
}

func (m *Mem) Delete(key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, has := m.files[key]
	if !has {
		return nil, fmt.Errorf("no key %q", key)
	}
	delete(m.files, key)
	return f.metadata, nil
}

func (m *Mem) Sync() error {
	return nil
}
