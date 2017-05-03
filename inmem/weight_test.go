package inmem

// Copyright: Boris Nagaev <bnagaev@gmail.com>

import (
	"testing"
)

func TestNewWeight(t *testing.T) {
	_, err := NewWeight(10, 1000)
	if err != nil {
		t.Errorf("NewWeight: %s", err)
	}
	_, err = NewWeight(-10, 1000)
	if err == nil {
		t.Errorf("NewWeight: want error, got no error")
	}
	_, err = NewWeight(10, -1000)
	if err == nil {
		t.Errorf("NewWeight: want error, got no error")
	}
}

func TestLen(t *testing.T) {
	c, err := NewWeight(10, 1000)
	if err != nil {
		t.Fatalf("NewWeight: %s", err)
	}
	if err := c.Add(1, "value", 1); err != nil {
		t.Fatalf("Add: %s", err)
	}
	if c.Len() != 1 {
		t.Errorf("Len: want 1, got %d", c.Len())
	}
}

func TestGet(t *testing.T) {
	c, err := NewWeight(10, 1000)
	if err != nil {
		t.Fatalf("NewWeight: %s", err)
	}
	if err := c.Add(1, "value", 1); err != nil {
		t.Fatalf("Add: %s", err)
	}
	v, has := c.Get(1)
	if !has {
		t.Fatalf("Get: no item for key %v", 1)
	}
	if vv, ok := v.(string); !ok || vv != "value" {
		t.Fatalf("Get: value of wrong type or value for key %v", 1)
	}
}

func TestEvict(t *testing.T) {
	cases := []struct {
		maxItems          int
		maxWeight, weight int64
	}{
		{10, 1000, 1},
		{100, 1000, 100},
	}
	for _, cs := range cases {
		c, err := NewWeight(cs.maxItems, cs.maxWeight)
		if err != nil {
			t.Fatalf("NewWeight: %s", err)
		}
		for key := 1; key <= 11; key++ {
			if err := c.Add(key, "value", cs.weight); err != nil {
				t.Fatalf("Add(%d): %s", key, err)
			}
		}
		if c.Len() != 10 {
			t.Errorf("Len: want 10, got %d", c.Len())
		}
		if _, has := c.Get(1); has {
			t.Errorf("Get(1): want to be evicted but not")
		}
		for key := 2; key <= 11; key++ {
			if _, has := c.Get(key); !has {
				t.Errorf("Get(%d): want but evicted", key)
			}
		}
	}
}
