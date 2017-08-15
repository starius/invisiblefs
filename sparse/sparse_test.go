package sparse

import (
	"bytes"
	"testing"
)

type DummyAppender []byte

func (a *DummyAppender) ReadAt(p []byte, off int64) (int, error) {
	if off < int64(len(*a)) {
		copy(p, (*a)[off:])
	}
	return len(p), nil
}

func (a *DummyAppender) Append(data []byte) (int, error) {
	*a = append(*a, data...)
	return len(data), nil
}

func (a *DummyAppender) Size() (int64, error) {
	return int64(len(*a)), nil
}

func TestSparse(t *testing.T) {
	data := &DummyAppender{}
	offsets := &DummyAppender{}
	s, err := NewSparse(data, offsets)
	if err != nil {
		t.Fatalf("NewSparse: %v", err)
	}
	buf := make([]byte, 10)
	if n, err := s.ReadAt(buf, 0); err != nil {
		t.Errorf("ReadAt: %v", err)
	} else if n != 10 {
		t.Errorf("ReadAt: n = %d", n)
	}
	buf2 := []byte{1,2,3,4,5}
	if n, err := s.WriteAt(buf2, 0); err != nil {
		t.Errorf("WriteAt: %v", err)
	} else if n != 5 {
		t.Errorf("WriteAt: n = %d", n)
	}
	buf3 := make([]byte, 5)
	if n, err := s.ReadAt(buf3, 0); err != nil {
		t.Errorf("ReadAt: %v", err)
	} else if n != 5 {
		t.Errorf("ReadAt: n = %d", n)
	}
	if !bytes.Equal(buf3, buf2) {
		t.Errorf("buf3 (%#v) != buf2 (%#v)", buf3, buf2)
	}
	if n, err := s.WriteAt(buf2, 3); err != nil {
		t.Errorf("WriteAt: %v", err)
	} else if n != 5 {
		t.Errorf("WriteAt: n = %d", n)
	}
	buf4 := make([]byte, 8)
	if n, err := s.ReadAt(buf4, 0); err != nil {
		t.Errorf("ReadAt: %v", err)
	} else if n != 8 {
		t.Errorf("ReadAt: n = %d", n)
	}
	buf4exp := []byte{1,2,3,1,2,3,4,5}
	if !bytes.Equal(buf4, buf4exp) {
		t.Errorf("buf4 (%#v) != buf4exp (%#v)", buf4, buf4exp)
	}
}

func TestAppend(t *testing.T) {
	data := &DummyAppender{}
	offsets := &DummyAppender{}
	s, err := NewSparse(data, offsets)
	if err != nil {
		t.Fatalf("NewSparse: %v", err)
	}
	for i := 0; i < 100; i++ {
		buf := make([]byte, 10)
		for j := range buf {
			buf[j] = byte(i)
		}
		if n, err := s.WriteAt(buf, int64(i * 10)); err != nil {
			t.Errorf("WriteAt: %v", err)
		} else if n != 10 {
			t.Errorf("WriteAt: n = %d", n)
		}
	}
	buf := make([]byte, 1000)
	if n, err := s.ReadAt(buf, 0); err != nil {
		t.Errorf("ReadAt: %v", err)
	} else if n != 1000 {
		t.Errorf("ReadAt: n = %d", n)
	}
	bufexp := make([]byte, 1000)
	for i := 0; i < 100; i++ {
		for j := 0; j < 10; j++ {
			bufexp[i*10 +j] = byte(i)
		}
	}
	if !bytes.Equal(buf, bufexp) {
		t.Errorf("buf (%#v) != bufexp (%#v)", buf, bufexp)
	}
}

func TestReopen(t *testing.T) {
	data := &DummyAppender{}
	offsets := &DummyAppender{}
	s, err := NewSparse(data, offsets)
	if err != nil {
		t.Fatalf("NewSparse: %v", err)
	}
	// Write concentric slices.
	for i := 0; i < 5; i++ {
		buf := make([]byte, 10-2*i)
		for j := 0; j < len(buf); j++ {
			buf[j] = byte(i)
		}
		if n, err := s.WriteAt(buf, int64(i)); err != nil {
			t.Errorf("WriteAt: %v", err)
		} else if n != len(buf) {
			t.Errorf("WriteAt: n = %d", n)
		}
	}
	// Reopen.
	s1, err := NewSparse(data, offsets)
	if err != nil {
		t.Fatalf("NewSparse: %v", err)
	}
	buf := make([]byte, 10)
	if n, err := s1.ReadAt(buf, 0); err != nil {
		t.Errorf("ReadAt: %v", err)
	} else if n != 10 {
		t.Errorf("ReadAt: n = %d", n)
	}
	bufexp := []byte{0, 1, 2, 3, 4, 4, 3, 2, 1, 0}
	if !bytes.Equal(buf, bufexp) {
		t.Errorf("buf (%#v) != bufexp (%#v)", buf, bufexp)
	}
}

func TestReadZeros(t *testing.T) {
	data := &DummyAppender{}
	offsets := &DummyAppender{}
	s, err := NewSparse(data, offsets)
	if err != nil {
		t.Fatalf("NewSparse: %v", err)
	}
	buf := []byte{1,2,3,4,5}
	if n, err := s.ReadAt(buf, 0); err != nil {
		t.Errorf("ReadAt: %v", err)
	} else if n != len(buf) {
		t.Errorf("ReadAt: n = %d", n)
	}
	bufexp := []byte{0,0,0,0,0}
	if !bytes.Equal(buf, bufexp) {
		t.Errorf("buf (%#v) != bufexp (%#v)", buf, bufexp)
	}
}
