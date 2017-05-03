package inmem

// Copyright: Boris Nagaev <bnagaev@gmail.com>

import (
	"container/list"
	"fmt"
	"sync"
)

// weightEntry is an entry in the cache.
type weightEntry struct {
	key    interface{}
	value  interface{}
	weight int64
}

// WeightCache provides LRU cache with weights support.
type WeightCache struct {
	maxItems          int
	maxWeight, weight int64
	lru               *list.List
	dict              map[interface{}]*list.Element
	m                 sync.Mutex
}

// NewWeight constructs a new WeightCache of the given number of items
// and the given maximum sum of weights of elements.
func NewWeight(maxItems int, maxWeight int64) (*WeightCache, error) {
	if maxItems <= 0 {
		return nil, fmt.Errorf("NewCache: maxItems <= 0")
	}
	if maxWeight <= 0 {
		return nil, fmt.Errorf("NewCache: maxWeight <= 0")
	}
	return &WeightCache{
		maxItems:  maxItems,
		maxWeight: maxWeight,
		lru:       list.New(),
		dict:      make(map[interface{}]*list.Element),
	}, nil
}

func (c *WeightCache) Len() int {
	c.m.Lock()
	defer c.m.Unlock()
	return c.lru.Len()
}

func (c *WeightCache) removeItem(e *list.Element) {
	c.lru.Remove(e)
	v := e.Value.(*weightEntry)
	c.weight -= v.weight
	delete(c.dict, v.key)
}

func (c *WeightCache) Add(key, value interface{}, weight int64) error {
	c.m.Lock()
	defer c.m.Unlock()
	if weight > c.maxWeight {
		return fmt.Errorf("too heavy item: %d > %d", weight, c.maxWeight)
	}
	if item, ok := c.dict[key]; ok {
		// Update existing entry.
		c.lru.MoveToFront(item)
		v := item.Value.(*weightEntry)
		c.weight += weight - v.weight
		v.weight = weight
		v.value = value
	} else {
		// Add new entry.
		c.dict[key] = c.lru.PushFront(&weightEntry{
			key:    key,
			value:  value,
			weight: weight,
		})
		c.weight += weight
	}
	// Remove old items.
	for c.lru.Len() > c.maxItems || c.weight > c.maxWeight {
		item := c.lru.Back()
		if item != nil {
			c.removeItem(item)
		}
	}
	return nil
}

func (c *WeightCache) Get(key interface{}) (interface{}, bool) {
	c.m.Lock()
	defer c.m.Unlock()
	if item, ok := c.dict[key]; ok {
		v := item.Value.(*weightEntry)
		return v.value, true
	}
	return nil, false
}

func (c *WeightCache) Remove(key interface{}) {
	c.m.Lock()
	defer c.m.Unlock()
	if item, ok := c.dict[key]; ok {
		c.removeItem(item)
	}
}
