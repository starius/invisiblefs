package tests

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/starius/invisiblefs/zipkvserver/kv"
)

func TestEmpty(t *testing.T, k kv.KV) {
	if has, _, err := k.Has("file"); err != nil {
		t.Errorf("k.Has: %s.", err)
	} else if has != false {
		t.Errorf("k.Has returned %#v, want false.", has)
	}
	if _, _, err := k.Get("file"); err == nil {
		t.Errorf("k.Get returned no error for absent file.")
	}
	if _, _, err := k.GetAt("file", 1, 2); err == nil {
		t.Errorf("k.GetAt returned no error for absent file.")
	}
}

func TestPut(t *testing.T, k kv.KV) {
	data0 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	if err := k.Put("file", data0, nil); err != nil {
		t.Fatalf("k.Put: %s.", err)
	}
	if has, _, err := k.Has("file"); err != nil {
		t.Errorf("k.Has: %s.", err)
	} else if has != true {
		t.Errorf("k.Has returned %#v, want true.", has)
	}
	if data, _, err := k.Get("file"); err != nil {
		t.Errorf("k.Get: %s.", err)
	} else if !bytes.Equal(data, data0) {
		t.Errorf("k.Get returned %#v, want %#v.", data, data0)
	}
	if data, _, err := k.GetAt("file", 1, 2); err != nil {
		t.Errorf("k.GetAt: %s.", err)
	} else if !bytes.Equal(data, data0[1:1+2]) {
		t.Errorf("k.GetAt returned %#v, want %#v.", data, data0[1:2])
	}
}

func TestPutLarge(t *testing.T, k kv.KV) {
	data0 := make([]byte, 10*1000*1000)
	// Put fibonacci numbers to the buffer.
	a, b := 0, 1
	for i := 0; i < len(data0); i++ {
		data0[i] = byte(a)
		a, b = b, (a+b)%256
	}
	if err := k.Put("file", data0, nil); err != nil {
		t.Fatalf("k.Put: %s.", err)
	}
	if has, _, err := k.Has("file"); err != nil {
		t.Errorf("k.Has: %s.", err)
	} else if has != true {
		t.Errorf("k.Has returned %#v, want true.", has)
	}
	if data, _, err := k.Get("file"); err != nil {
		t.Errorf("k.Get: %s.", err)
	} else if !bytes.Equal(data, data0) {
		t.Errorf("k.Get returned %#v, want %#v.", data, data0)
	}
	data0s := data0[10000 : 10000+20000]
	if data, _, err := k.GetAt("file", 10000, 20000); err != nil {
		t.Errorf("k.GetAt: %s.", err)
	} else if !bytes.Equal(data, data0s) {
		t.Errorf("k.GetAt returned %#v, want %#v.", data, data0s)
	}
}

func TestPutMany1(t *testing.T, k kv.KV, n int) {
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("file%d", i)
		data0 := make([]byte, 1000)
		// Put numbers to the buffer.
		for j := 0; j < len(data0); j++ {
			data0[j] = byte((i + j) % 256)
		}
		if err := k.Put(key, data0, nil); err != nil {
			t.Fatalf("k.Put(%q): %s.", key, err)
		}
	}
}

func TestPutMany2(t *testing.T, k kv.KV, n int) {
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("file%d", i)
		data0 := make([]byte, 1000)
		// Put numbers to the buffer.
		for j := 0; j < len(data0); j++ {
			data0[j] = byte((i + j) % 256)
		}
		if has, _, err := k.Has(key); err != nil {
			t.Errorf("k.Has: %s.", err)
		} else if has != true {
			t.Errorf("k.Has returned %#v, want true.", has)
		}
		if data, _, err := k.Get(key); err != nil {
			t.Errorf("k.Get: %s.", err)
		} else if !bytes.Equal(data, data0) {
			t.Errorf("k.Get returned %#v, want %#v.", data, data0)
		}
		data0s := data0[100 : 100+200]
		if data, _, err := k.GetAt(key, 100, 200); err != nil {
			t.Errorf("k.GetAt: %s.", err)
		} else if !bytes.Equal(data, data0s) {
			t.Errorf("k.GetAt returned %#v, want %#v.", data, data0s)
		}
	}
}

func TestPutMany(t *testing.T, k kv.KV) {
	n := 10 * 1000
	TestPutMany1(t, k, n)
	TestPutMany2(t, k, n)
}

func TestDelete(t *testing.T, k kv.KV) {
	data0 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	if err := k.Put("file", data0, nil); err != nil {
		t.Fatalf("k.Put: %s.", err)
	}
	if _, err := k.Delete("file"); err != nil {
		t.Fatalf("k.Delete: %s.", err)
	}
	if has, _, err := k.Has("file"); err != nil {
		t.Errorf("k.Has: %s.", err)
	} else if has != false {
		t.Errorf("k.Has returned %#v, want false.", has)
	}
	if _, _, err := k.Get("file"); err == nil {
		t.Errorf("k.Get returned no error for absent file.")
	}
	if _, _, err := k.GetAt("file", 1, 2); err == nil {
		t.Errorf("k.GetAt returned no error for absent file.")
	}
}
