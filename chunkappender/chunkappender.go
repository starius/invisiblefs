package chunkappender

import (
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/starius/invisiblefs/inmem"
)

type ChunkStore interface {
	Sizes() ([]int64, error)
	Get(i int) ([]byte, error)
	Put(i int, data []byte) error
}

type ChunkAppender struct {
	backend ChunkStore

	ends   []int64
	endsMu sync.RWMutex

	writeMu sync.Mutex

	cache *inmem.WeightCache
}

// NewChunkAppender creates ChunkAppender from ChunkStore.
func NewChunkAppender(backend ChunkStore, cacheMB int) (*ChunkAppender, error) {
	sizes, err := backend.Sizes()
	if err != nil {
		return nil, fmt.Errorf("backendt.Sizes: %s", err)
	}
	ends := make([]int64, len(sizes))
	var sum int64
	for i, v := range sizes {
		sum += v
		ends[i] = sum
	}
	maxItems := 1024
	maxWeight := int64(cacheMB) * 1024 * 1024
	cache, err := inmem.NewWeight(maxItems, maxWeight)
	if err != nil {
		return nil, fmt.Errorf("inmem.NewWeight: %s", err)
	}
	return &ChunkAppender{
		backend: backend,
		ends:    ends,
		cache:   cache,
	}, nil
}

func (c *ChunkAppender) c2b(c1, c2 int64) (b1, b2 int) {
	c.endsMu.RLock()
	defer c.endsMu.RUnlock()
	b1 = sort.Search(len(c.ends), func(i int) bool {
		return c.ends[i] > c1
	})
	b2 = b1 + 1 + sort.Search(len(c.ends)-b1, func(i int) bool {
		return c.ends[b1+i] >= c2
	})
	return
}

func (c *ChunkAppender) b2c(b int) (c1, c2 int64) {
	c.endsMu.RLock()
	defer c.endsMu.RUnlock()
	c1 = 0
	if b > 0 {
		c1 = c.ends[b-1]
	}
	c2 = c.ends[b]
	return
}

func (c *ChunkAppender) read(b int) ([]byte, error) {
	if data, has := c.cache.Get(b); has {
		return data.([]byte), nil
	}
	data, err := c.backend.Get(b)
	if err != nil {
		return nil, fmt.Errorf("backend.Get: %s", err)
	}
	if err := c.cache.Add(b, data, int64(len(data))); err != nil {
		log.Printf("Can't store chunk %d in cache: %s.", b, err)
	}
	return data, nil
}

func (c *ChunkAppender) sizes() (int, int64) {
	c.endsMu.RLock()
	defer c.endsMu.RUnlock()
	if len(c.ends) == 0 {
		return 0, 0
	} else {
		return len(c.ends), c.ends[len(c.ends)-1]
	}
}

// ReadAt reads len(b) bytes from the file starting at byte offset off.
func (c *ChunkAppender) ReadAt(p []byte, off int64) (int, error) {
	end := off + int64(len(p))
	b1, b2 := c.c2b(off, end)
	var err0 error
	var err0m sync.Mutex
	var wg sync.WaitGroup
	for b := b1; b < b2; b++ {
		wg.Add(1)
		go func(b int) {
			defer wg.Done()
			data, err := c.read(b)
			if err != nil {
				err0m.Lock()
				defer err0m.Unlock()
				err0 = err
				return
			}
			x, y := c.b2c(b)
			if x < off {
				data = data[off-x:]
				x = off
			}
			if end < y {
				data = data[:int64(len(data))-(y-end)]
				y = end
			}
			copy(p[x-off:y-off], data)
		}(b)
	}
	wg.Wait()
	if err0 != nil {
		return 0, err0
	}
	return len(p), nil
}

// WriteAt writes len(p) bytes from p, adding them to the backend.
func (c *ChunkAppender) WriteAt(p []byte, off int64) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	nblocks, nbytes := c.sizes()
	if off != nbytes {
		return 0, fmt.Errorf("attempt to perform non-append write at %d. File size is %d.", off, nbytes)
	}
	newBlock := nblocks
	if err := c.backend.Put(newBlock, p); err != nil {
		return 0, fmt.Errorf("backend.Put(%d) failed: %s", newBlock, err)
	}
	c.endsMu.Lock()
	defer c.endsMu.Unlock()
	c.ends = append(c.ends, nbytes+int64(len(p)))
	return len(p), nil
}

type fileInfo int64

func (f fileInfo) Name() string {
	return "file"
}

func (f fileInfo) Size() int64 {
	return int64(f)
}

func (f fileInfo) Mode() os.FileMode {
	return 0600
}

func (f fileInfo) ModTime() time.Time {
	return time.Unix(0, 0)
}

func (f fileInfo) IsDir() bool {
	return false
}

func (f fileInfo) Sys() interface{} {
	return nil
}

func (c *ChunkAppender) Stat() (os.FileInfo, error) {
	_, nbytes := c.sizes()
	return fileInfo(nbytes), nil
}

func (c *ChunkAppender) Truncate(size int64) error {
	_, nbytes := c.sizes()
	if size < nbytes {
		return fmt.Errorf("can't shrink the file")
	} else {
		_, err := c.WriteAt(make([]byte, size-nbytes), nbytes)
		return err
	}
}
