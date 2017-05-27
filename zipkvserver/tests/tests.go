package tests

import (
	"bytes"
	"testing"

	"github.com/starius/invisiblefs/zipkvserver/zipkv"
)

func TestEmpty(t *testing.T, kv zipkv.KV) {
	if has, _, err := kv.Has("file"); err != nil {
		t.Errorf("kv.Has: %s.", err)
	} else if has != false {
		t.Errorf("kv.Has returned %#v, want false.", has)
	}
	if _, _, err := kv.Get("file"); err == nil {
		t.Errorf("kv.Get returned no error for absent file.")
	}
	if _, _, err := kv.GetAt("file", 1, 2); err == nil {
		t.Errorf("kv.GetAt returned no error for absent file.")
	}
}

func TestPut(t *testing.T, kv zipkv.KV) {
	data0 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	if err := kv.Put("file", data0, nil); err != nil {
		t.Fatalf("kv.Put: %s.", err)
	}
	if has, _, err := kv.Has("file"); err != nil {
		t.Errorf("kv.Has: %s.", err)
	} else if has != true {
		t.Errorf("kv.Has returned %#v, want true.", has)
	}
	if data, _, err := kv.Get("file"); err != nil {
		t.Errorf("kv.Get: %s.", err)
	} else if !bytes.Equal(data, data0) {
		t.Errorf("kv.Get returned %#v, want %#v.", data, data0)
	}
	if data, _, err := kv.GetAt("file", 1, 2); err != nil {
		t.Errorf("kv.GetAt: %s.", err)
	} else if !bytes.Equal(data, data0[1:1+2]) {
		t.Errorf("kv.GetAt returned %#v, want %#v.", data, data0[1:2])
	}
}

func TestPutLarge(t *testing.T, kv zipkv.KV) {
	data0 := make([]byte, 10*1000*1000)
	// Put fibonacci numbers to the buffer.
	a, b := 0, 1
	for i := 0; i < len(data0); i++ {
		data0[i] = byte(a)
		a, b = b, (a+b)%256
	}
	if err := kv.Put("file", data0, nil); err != nil {
		t.Fatalf("kv.Put: %s.", err)
	}
	if has, _, err := kv.Has("file"); err != nil {
		t.Errorf("kv.Has: %s.", err)
	} else if has != true {
		t.Errorf("kv.Has returned %#v, want true.", has)
	}
	if data, _, err := kv.Get("file"); err != nil {
		t.Errorf("kv.Get: %s.", err)
	} else if !bytes.Equal(data, data0) {
		t.Errorf("kv.Get returned %#v, want %#v.", data, data0)
	}
	data0s := data0[10000 : 10000+20000]
	if data, _, err := kv.GetAt("file", 10000, 20000); err != nil {
		t.Errorf("kv.GetAt: %s.", err)
	} else if !bytes.Equal(data, data0s) {
		t.Errorf("kv.GetAt returned %#v, want %#v.", data, data0s)
	}
}

func TestDelete(t *testing.T, kv zipkv.KV) {
	data0 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	if err := kv.Put("file", data0, nil); err != nil {
		t.Fatalf("kv.Put: %s.", err)
	}
	if _, err := kv.Delete("file"); err != nil {
		t.Fatalf("kv.Delete: %s.", err)
	}
	if has, _, err := kv.Has("file"); err != nil {
		t.Errorf("kv.Has: %s.", err)
	} else if has != false {
		t.Errorf("kv.Has returned %#v, want false.", has)
	}
	if _, _, err := kv.Get("file"); err == nil {
		t.Errorf("kv.Get returned no error for absent file.")
	}
	if _, _, err := kv.GetAt("file", 1, 2); err == nil {
		t.Errorf("kv.GetAt returned no error for absent file.")
	}
}
