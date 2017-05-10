package inmem

// Copyright: Boris Nagaev <bnagaev@gmail.com>

import (
	"container/list"
	"fmt"
	"sync"
)

// CloserEntry is an entry in the cache.
type CloserEntry struct {
	Key   interface{}
	Value interface{}
}

// CloserCache provides LRU cache with closer support.
type CloserCache struct {
	maxItems int
	lru      *list.List
	dict     map[interface{}]*list.Element
	m        sync.Mutex
	closer   func(interface{})
}

// NewCloserCache constructs a new CloserCache of the given number of items.
// and the given closer.
func NewCloserCache(maxItems int, closer func(interface{})) (*CloserCache, error) {
	if maxItems <= 0 {
		return nil, fmt.Errorf("NewCloserCache: maxItems <= 0")
	}
	return &CloserCache{
		maxItems: maxItems,
		lru:      list.New(),
		dict:     make(map[interface{}]*list.Element),
		closer:   closer,
	}, nil
}

func (c *CloserCache) Len() int {
	c.m.Lock()
	defer c.m.Unlock()
	return c.lru.Len()
}

func (c *CloserCache) removeItem(e *list.Element) {
	c.lru.Remove(e)
	v := e.Value.(*CloserEntry)
	c.closer(v.Value)
	delete(c.dict, v.Key)
}

func (c *CloserCache) Add(key, value interface{}) {
	c.m.Lock()
	defer c.m.Unlock()
	if item, ok := c.dict[key]; ok {
		// Update existing entry.
		c.lru.MoveToFront(item)
		v := item.Value.(*CloserEntry)
		v.Value = value
	} else {
		// Add new entry.
		c.dict[key] = c.lru.PushFront(&CloserEntry{
			Key:   key,
			Value: value,
		})
	}
	// Remove old items.
	if c.lru.Len() > c.maxItems {
		item := c.lru.Back()
		if item != nil {
			c.removeItem(item)
		}
	}
}

func (c *CloserCache) Get(key interface{}) (interface{}, bool) {
	c.m.Lock()
	defer c.m.Unlock()
	if item, ok := c.dict[key]; ok {
		v := item.Value.(*CloserEntry)
		return v.Value, true
	}
	return nil, false
}

func (c *CloserCache) Remove(key interface{}) {
	c.m.Lock()
	defer c.m.Unlock()
	if item, ok := c.dict[key]; ok {
		c.removeItem(item)
	}
}

func (c *CloserCache) Items() []CloserEntry {
	c.m.Lock()
	defer c.m.Unlock()
	var ee []CloserEntry
	for _, item := range c.dict {
		ee = append(ee, *item.Value.(*CloserEntry))
	}
	return ee
}
