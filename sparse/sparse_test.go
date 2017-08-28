package sparse

import (
	"bytes"
	"fmt"
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
	s, err := NewSparse2(data, offsets)
	if err != nil {
		t.Fatalf("NewSparse2: %v", err)
	}
	buf := make([]byte, 10)
	if n, err := s.ReadAt(buf, 0); err != nil {
		t.Errorf("ReadAt: %v", err)
	} else if n != 10 {
		t.Errorf("ReadAt: n = %d", n)
	}
	buf2 := []byte{1, 2, 3, 4, 5}
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
	buf4exp := []byte{1, 2, 3, 1, 2, 3, 4, 5}
	if !bytes.Equal(buf4, buf4exp) {
		t.Errorf("buf4 (%#v) != buf4exp (%#v)", buf4, buf4exp)
	}
}

func TestAppend(t *testing.T) {
	data := &DummyAppender{}
	offsets := &DummyAppender{}
	s, err := NewSparse2(data, offsets)
	if err != nil {
		t.Fatalf("NewSparse2: %v", err)
	}
	for i := 0; i < 100; i++ {
		buf := make([]byte, 10)
		for j := range buf {
			buf[j] = byte(i)
		}
		if n, err := s.WriteAt(buf, int64(i*10)); err != nil {
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
			bufexp[i*10+j] = byte(i)
		}
	}
	if !bytes.Equal(buf, bufexp) {
		t.Errorf("buf (%#v) != bufexp (%#v)", buf, bufexp)
	}
}

func TestReopen(t *testing.T) {
	data := &DummyAppender{}
	offsets := &DummyAppender{}
	s, err := NewSparse2(data, offsets)
	if err != nil {
		t.Fatalf("NewSparse2: %v", err)
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
	s1, err := NewSparse2(data, offsets)
	if err != nil {
		t.Fatalf("NewSparse2: %v", err)
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
	// Write more.
	if n, err := s1.WriteAt([]byte{5, 5, 5, 5, 5}, 0); err != nil {
		t.Errorf("WriteAt: %v", err)
	} else if n != 5 {
		t.Errorf("WriteAt: n = %d", n)
	}
	buf2 := make([]byte, 10)
	if n, err := s1.ReadAt(buf2, 0); err != nil {
		t.Errorf("ReadAt: %v", err)
	} else if n != 10 {
		t.Errorf("ReadAt: n = %d", n)
	}
	bufexp2 := []byte{5, 5, 5, 5, 5, 4, 3, 2, 1, 0}
	if !bytes.Equal(buf2, bufexp2) {
		t.Errorf("buf2 (%#v) != bufexp2 (%#v)", buf2, bufexp2)
	}
	// Reopen one more time.
	s2, err := NewSparse2(data, offsets)
	if err != nil {
		t.Fatalf("NewSparse2: %v", err)
	}
	buf3 := make([]byte, 10)
	if n, err := s2.ReadAt(buf3, 0); err != nil {
		t.Errorf("ReadAt: %v", err)
	} else if n != 10 {
		t.Errorf("ReadAt: n = %d", n)
	}
	if !bytes.Equal(buf3, bufexp2) {
		t.Errorf("buf3 (%#v) != bufexp (%#v)", buf3, bufexp2)
	}
}

func TestReadZeros(t *testing.T) {
	data := &DummyAppender{}
	offsets := &DummyAppender{}
	s, err := NewSparse2(data, offsets)
	if err != nil {
		t.Fatalf("NewSparse2: %v", err)
	}
	buf := []byte{1, 2, 3, 4, 5}
	if n, err := s.ReadAt(buf, 0); err != nil {
		t.Errorf("ReadAt: %v", err)
	} else if n != len(buf) {
		t.Errorf("ReadAt: n = %d", n)
	}
	bufexp := []byte{0, 0, 0, 0, 0}
	if !bytes.Equal(buf, bufexp) {
		t.Errorf("buf (%#v) != bufexp (%#v)", buf, bufexp)
	}
}

func TestWrites(t *testing.T) {
	type Write struct {
		off  int64
		data []byte
	}
	cases := []struct {
		writes   []Write
		readOff  int64
		expected []byte
	}{
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  2,
					data: []byte{2, 2, 2},
				},
			},
			readOff:  0,
			expected: []byte{0, 0, 2, 2, 2, 1, 1, 1},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  2,
					data: []byte{2, 2},
				},
			},
			readOff:  0,
			expected: []byte{0, 0, 2, 2, 0, 1, 1, 1},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  2,
					data: []byte{2, 2, 2, 2},
				},
			},
			readOff:  0,
			expected: []byte{0, 0, 2, 2, 2, 2, 1, 1},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  2,
					data: []byte{2, 2, 2, 2, 2, 2},
				},
			},
			readOff:  0,
			expected: []byte{0, 0, 2, 2, 2, 2, 2, 2},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  2,
					data: []byte{2, 2, 2, 2, 2, 2, 2},
				},
			},
			readOff:  0,
			expected: []byte{0, 0, 2, 2, 2, 2, 2, 2, 2},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  5,
					data: []byte{2, 2, 2},
				},
			},
			readOff:  0,
			expected: []byte{0, 0, 0, 0, 0, 2, 2, 2, 0},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  5,
					data: []byte{2, 2, 2, 2, 2},
				},
			},
			readOff:  0,
			expected: []byte{0, 0, 0, 0, 0, 2, 2, 2, 2},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  6,
					data: []byte{2, 2, 2, 2, 2},
				},
			},
			readOff:  4,
			expected: []byte{0, 1, 2, 2, 2},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  8,
					data: []byte{2, 2},
				},
			},
			readOff:  4,
			expected: []byte{0, 1, 1, 1, 2, 2, 0},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  9,
					data: []byte{2, 2},
				},
			},
			readOff:  4,
			expected: []byte{0, 1, 1, 1, 0, 2, 2, 0},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  9,
					data: []byte{2, 2},
				},
				{
					off:  7,
					data: []byte{3, 3, 3},
				},
			},
			readOff:  4,
			expected: []byte{0, 1, 1, 3, 3, 3, 2, 0},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  9,
					data: []byte{2, 2},
				},
				{
					off:  5,
					data: []byte{3, 3, 3, 3, 3, 3},
				},
			},
			readOff:  4,
			expected: []byte{0, 3, 3, 3, 3, 3, 3, 0},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  9,
					data: []byte{2, 2},
				},
				{
					off:  6,
					data: []byte{3, 3, 3, 3, 3},
				},
			},
			readOff:  4,
			expected: []byte{0, 1, 3, 3, 3, 3, 3, 0},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  9,
					data: []byte{2, 2},
				},
				{
					off:  5,
					data: []byte{3, 3, 3, 3, 3},
				},
			},
			readOff:  4,
			expected: []byte{0, 3, 3, 3, 3, 3, 2, 0},
		},
		{
			writes: []Write{
				{
					off:  5,
					data: []byte{1, 1, 1},
				},
				{
					off:  9,
					data: []byte{2, 2},
				},
				{
					off:  4,
					data: []byte{3, 3, 3, 3, 3, 3, 3},
				},
			},
			readOff:  3,
			expected: []byte{0, 3, 3, 3, 3, 3, 3, 3, 0},
		},
	}
	for _, c := range cases {
		data := &DummyAppender{}
		offsets := &DummyAppender{}
		s, err := NewSparse2(data, offsets)
		if err != nil {
			t.Fatalf("NewSparse2: %v", err)
		}
		for _, write := range c.writes {
			if n, err := s.WriteAt(write.data, write.off); err != nil {
				t.Errorf("WriteAt: %v", err)
			} else if n != len(write.data) {
				t.Errorf("WriteAt: n = %d", n)
			}
		}
		buf := make([]byte, len(c.expected))
		if n, err := s.ReadAt(buf, c.readOff); err != nil {
			t.Errorf("ReadAt: %v", err)
		} else if n != len(c.expected) {
			t.Errorf("ReadAt: n = %d", n)
		}
		if !bytes.Equal(buf, c.expected) {
			t.Errorf("buf (%#v) != bufexp (%#v)", buf, c.expected)
		}
		// Reopen.
		s1, err := NewSparse2(data, offsets)
		if err != nil {
			t.Fatalf("NewSparse2: %v", err)
		}
		buf2 := make([]byte, len(c.expected))
		if n, err := s1.ReadAt(buf2, c.readOff); err != nil {
			t.Errorf("ReadAt: %v", err)
		} else if n != len(c.expected) {
			t.Errorf("ReadAt: n = %d", n)
		}
		if !bytes.Equal(buf2, c.expected) {
			t.Errorf("buf2 (%#v) != bufexp (%#v)", buf2, c.expected)
		}
	}
}

type BrokenAppender struct {
	err  error
	impl Appender
}

func (a *BrokenAppender) ReadAt(p []byte, off int64) (int, error) {
	if a.err != nil {
		return 0, a.err
	}
	return a.impl.ReadAt(p, off)
}

func (a *BrokenAppender) Append(data []byte) (int, error) {
	if a.err != nil {
		return 0, a.err
	}
	return a.impl.Append(data)
}

func (a *BrokenAppender) Size() (int64, error) {
	if a.err != nil {
		return 0, a.err
	}
	return a.impl.Size()
}

func TestError(t *testing.T) {
	cases := []struct {
		breakData, breakOffsets bool
	}{
		{true, false},
		{false, true},
		{true, true},
	}
	for _, tc := range cases {
		data := &BrokenAppender{impl: &DummyAppender{}}
		offsets := &BrokenAppender{impl: &DummyAppender{}}
		s, err := NewSparse2(data, offsets)
		if err != nil {
			t.Fatalf("NewSparse2: %v", err)
		}
		// Write something.
		buf := []byte{1, 2, 3}
		if n, err := s.WriteAt(buf, 0); err != nil {
			t.Errorf("WriteAt: %v", err)
		} else if n != len(buf) {
			t.Errorf("WriteAt: n = %d", n)
		}
		// Break it and make sure write breaks.
		if tc.breakData {
			data.err = fmt.Errorf("test error for data")
		}
		if tc.breakOffsets {
			offsets.err = fmt.Errorf("test error for offsets")
		}
		if _, err := s.WriteAt(buf, 0); err == nil {
			t.Errorf("WriteAt: want error")
		}
		// Fix appenders and make sure write still fails.
		data.err = nil
		offsets.err = nil
		if _, err := s.WriteAt(buf, 0); err == nil {
			t.Errorf("WriteAt: want error")
		}
		// Make sure read still works.
		buf2 := make([]byte, 3)
		if n, err := s.ReadAt(buf2, 0); err != nil {
			t.Errorf("ReadAt: %v", err)
		} else if n != 3 {
			t.Errorf("ReadAt: n = %d", n)
		}
		if !bytes.Equal(buf2, buf) {
			t.Errorf("buf (%#v) != bufexp (%#v)", buf2, buf)
		}
	}
}

func TestFailsIfExtraData(t *testing.T) {
	data := &DummyAppender{}
	offsets := &DummyAppender{}
	s, err := NewSparse2(data, offsets)
	if err != nil {
		t.Fatalf("NewSparse2: %v", err)
	}
	// Write something.
	if n, err := s.WriteAt([]byte{1, 2, 3}, 0); err != nil {
		t.Fatalf("WriteAt: %v", err)
	} else if n != 3 {
		t.Fatalf("WriteAt: n = %d", n)
	}
	// Write something directly to data.
	if n, err := data.Append([]byte{4, 5}); err != nil {
		t.Fatalf("Append: %v", err)
	} else if n != 2 {
		t.Fatalf("Append: n = %d", n)
	}
	if _, err = NewSparse2(data, offsets); err == nil {
		t.Errorf("want error")
	}
}

func TestFailsIfDataCut(t *testing.T) {
	data := &DummyAppender{}
	offsets := &DummyAppender{}
	s, err := NewSparse2(data, offsets)
	if err != nil {
		t.Fatalf("NewSparse2: %v", err)
	}
	// Write something.
	if n, err := s.WriteAt([]byte{1, 2, 3}, 0); err != nil {
		t.Fatalf("WriteAt: %v", err)
	} else if n != 3 {
		t.Fatalf("WriteAt: n = %d", n)
	}
	data2 := &DummyAppender{1, 2}
	if _, err = NewSparse2(data2, offsets); err == nil {
		t.Errorf("want error")
	}
}
