package fskv

import (
	"os"
	"testing"

	"github.com/starius/invisiblefs/zipkvserver/tests"
)

func TestEmpty(t *testing.T) {
	dir := os.TempDir()
	defer os.RemoveAll(dir)
	kv, err := New(dir)
	if err != nil {
		t.Fatalf("Failed to create fskv: %s.", err)
	}
	tests.TestEmpty(t, kv)
}

func TestPut(t *testing.T) {
	dir := os.TempDir()
	defer os.RemoveAll(dir)
	kv, err := New(dir)
	if err != nil {
		t.Fatalf("Failed to create fskv: %s.", err)
	}
	tests.TestPut(t, kv)
}

func TestPutLarge(t *testing.T) {
	dir := os.TempDir()
	defer os.RemoveAll(dir)
	kv, err := New(dir)
	if err != nil {
		t.Fatalf("Failed to create fskv: %s.", err)
	}
	tests.TestPutLarge(t, kv)
}

func TestPutMany(t *testing.T) {
	dir := os.TempDir()
	defer os.RemoveAll(dir)
	kv, err := New(dir)
	if err != nil {
		t.Fatalf("Failed to create fskv: %s.", err)
	}
	tests.TestPutMany(t, kv)
}

func TestDelete(t *testing.T) {
	dir := os.TempDir()
	defer os.RemoveAll(dir)
	kv, err := New(dir)
	if err != nil {
		t.Fatalf("Failed to create fskv: %s.", err)
	}
	tests.TestDelete(t, kv)
}
