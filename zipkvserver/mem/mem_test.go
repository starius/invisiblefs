package mem

import (
	"testing"

	"github.com/starius/invisiblefs/zipkvserver/tests"
)

func TestEmpty(t *testing.T) {
	kv, err := New()
	if err != nil {
		t.Fatalf("Failed to create mem: %s.", err)
	}
	tests.TestEmpty(t, kv)
}

func TestPut(t *testing.T) {
	kv, err := New()
	if err != nil {
		t.Fatalf("Failed to create mem: %s.", err)
	}
	tests.TestPut(t, kv)
}

func TestPutLarge(t *testing.T) {
	kv, err := New()
	if err != nil {
		t.Fatalf("Failed to create mem: %s.", err)
	}
	tests.TestPutLarge(t, kv)
}

func TestPutMany(t *testing.T) {
	kv, err := New()
	if err != nil {
		t.Fatalf("Failed to create mem: %s.", err)
	}
	tests.TestPutMany(t, kv)
}

func TestDelete(t *testing.T) {
	kv, err := New()
	if err != nil {
		t.Fatalf("Failed to create mem: %s.", err)
	}
	tests.TestDelete(t, kv)
}
