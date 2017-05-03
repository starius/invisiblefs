package chunkappender

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

type DummyBackend struct {
	m      sync.Mutex
	chunks [][]byte
}

func (d *DummyBackend) Sizes() ([]int64, error) {
	d.m.Lock()
	defer d.m.Unlock()
	sizes := make([]int64, len(d.chunks))
	for i, c := range d.chunks {
		sizes[i] = int64(len(c))
	}
	return sizes, nil
}

func (d *DummyBackend) Get(b int) ([]byte, error) {
	d.m.Lock()
	defer d.m.Unlock()
	if b < 0 || b > len(d.chunks) {
		return nil, fmt.Errorf("out of range: %d not in [%d, %d)", b, 0, len(d.chunks))
	}
	return d.chunks[b], nil
}

func (d *DummyBackend) Put(i int, data []byte) error {
	d.m.Lock()
	defer d.m.Unlock()
	if i != len(d.chunks) {
		return fmt.Errorf("not append: i=%d, len(d.chunks)=%d", i, len(d.chunks))
	}
	d.chunks = append(d.chunks, data)
	return nil
}

func TestRead(t *testing.T) {
	text := []byte("hello world")
	be := &DummyBackend{
		chunks: [][]byte{
			text[0:5],
			text[5:6],
			text[6:11],
		},
	}
	ca, err := NewChunkAppender(be, 1)
	if err != nil {
		t.Fatalf("NewChunkAppender: %s", err)
	}
	buf0 := make([]byte, len(text))
	for x := 0; x < len(text); x++ {
		for y := x; y <= len(text); y++ {
			buf := buf0[:y-x]
			n, err := ca.ReadAt(buf, int64(x))
			if err != nil {
				t.Errorf("reading [%d, %d): %s", x, y, err)
			}
			if n != len(buf) {
				t.Errorf("reading [%d, %d): want %d bytes, got %s", x, y, len(buf), n)
			}
			if !bytes.Equal(buf, text[x:y]) {
				t.Errorf("ReadAt [%d, %d) returned %q, want %q", x, y, string(buf), string(text[x:y]))
			}
		}
	}
}

// createLargeText returns "0123..." of length about n.
func createLargeText(n int) []byte {
	text := make([]byte, n)
	b := bytes.NewBuffer(text[:0])
	i := 0
	for b.Len() < n {
		fmt.Fprintf(b, "%d", i)
		i++
	}
	// Strip zeros from the end.
	for len(text) > 0 && text[len(text)-1] == '\x00' {
		text = text[:len(text)-1]
	}
	return text
}

func TestReadLarge(t *testing.T) {
	text := createLargeText(10 * 1000 * 1000)
	var chunks [][]byte
	// Fill chunks.
	t1 := text
	for len(t1) > 0 {
		l := 500000
		if l > len(t1) {
			l = len(t1)
		}
		chunks = append(chunks, t1[:l])
		t1 = t1[l:]
	}
	be := &DummyBackend{
		chunks: chunks,
	}
	ca, err := NewChunkAppender(be, 3)
	if err != nil {
		t.Fatalf("NewChunkAppender: %s", err)
	}
	buf0 := make([]byte, len(text))
	for x := 0; x < len(text); x += 100000 {
		for y := x; y <= len(text); y += 100000 {
			buf := buf0[:y-x]
			n, err := ca.ReadAt(buf, int64(x))
			if err != nil {
				t.Errorf("reading [%d, %d): %s", x, y, err)
			}
			if n != len(buf) {
				t.Errorf("reading [%d, %d): want %d bytes, got %s", x, y, len(buf), n)
			}
			if !bytes.Equal(buf, text[x:y]) {
				t.Errorf("ReadAt [%d, %d) returned %q, want %q", x, y, string(buf), string(text[x:y]))
			}
		}
	}
}

func TestWrite(t *testing.T) {
	text := createLargeText(10 * 1000 * 1000)
	be := &DummyBackend{}
	ca, err := NewChunkAppender(be, 3)
	if err != nil {
		t.Fatalf("NewChunkAppender: %s", err)
	}
	t1 := text
	var total int64
	for len(t1) > 0 {
		l := 500000
		if l > len(t1) {
			l = len(t1)
		}
		chunk := t1[:l]
		t1 = t1[l:]
		nextTotal := total + int64(len(chunk))
		n, err := ca.WriteAt(chunk, total)
		if err != nil {
			t.Errorf("writing [%d, %d): %s", total, nextTotal, err)
		}
		if n != len(chunk) {
			t.Errorf("writing [%d, %d): want %d bytes, got %s", total, nextTotal, len(chunk), n)
		}
		total = nextTotal
	}
	t2 := make([]byte, len(text))
	n, err := ca.ReadAt(t2, 0)
	if err != nil {
		t.Errorf("reading [%d, %d): %s", 0, len(t2), err)
	}
	if n != len(t2) {
		t.Errorf("reading [%d, %d): want %d bytes, got %s", 0, len(t2), len(t2), n)
	}
	if !bytes.Equal(t2, text) {
		t.Errorf("ReadAt [%d, %d) returned %q, want %q", 0, len(t2), string(t2), string(text))
	}
}
