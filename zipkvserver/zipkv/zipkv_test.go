package zipkv

import (
	"fmt"
	"testing"

	"github.com/starius/invisiblefs/zipkvserver/mem"
	"github.com/starius/invisiblefs/zipkvserver/tests"
)

func instance(size int) (*Frontend, error) {
	m, err := mem.New()
	if err != nil {
		return nil, fmt.Errorf("Failed to create mem: %s.", err)
	}
	return Zip(m, size, -1)
}

func TestEmpty(t *testing.T) {
	kv, err := instance(2*1000*1000)
	if err != nil {
		t.Fatalf("Failed to create Frontend: %s.", err)
	}
	tests.TestEmpty(t, kv)
}

func TestPut(t *testing.T) {
	kv, err := instance(2*1000*1000)
	if err != nil {
		t.Fatalf("Failed to create Frontend: %s.", err)
	}
	tests.TestPut(t, kv)
}

func TestPutLarge(t *testing.T) {
	kv, err := instance(40*1000*1000)
	if err != nil {
		t.Fatalf("Failed to create Frontend: %s.", err)
	}
	tests.TestPutLarge(t, kv)
}

func TestPutMany(t *testing.T) {
	kv, err := instance(2*1000*1000)
	if err != nil {
		t.Fatalf("Failed to create Frontend: %s.", err)
	}
	tests.TestPutMany(t, kv)
}

func TestDelete(t *testing.T) {
	kv, err := instance(2*1000*1000)
	if err != nil {
		t.Fatalf("Failed to create Frontend: %s.", err)
	}
	tests.TestDelete(t, kv)
}
