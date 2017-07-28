package sparse

import (
	"bytes"
	"encoding/binary"
	"io"
	"runtime"
)

//go:generate g++ -std=c++11 cpp/index.cpp -c -o /tmp/index.o

// #cgo CFLAGS: -I${SRCDIR}
// #cgo LDFLAGS: /tmp/index.o -lstdc++
// #include "cpp/index.h"
import "C"

type Appender interface {
	io.ReaderAt
	Append(data []byte) (int, error)
	Size() (int64, error)
}

type AppenderSet interface {
	List() (map[string]Appender, error)
	Open(name string) (Appender, error)
	Remove(name string) error
}

type Sparse struct {
	index         *C.Index
	data, offsets Appender
}

type byteReader struct {
	a    Appender
	i, n int64
	buf  []byte
}

func (r *byteReader) ReadByte() (byte, error) {
	if r.i >= r.n {
		return 0, io.EOF
	}
	if n, err := r.a.ReadAt(r.buf, r.i); err != nil {
		return 0, err
	} else if n != 1 {
		return 0, io.EOF
	}
	r.i++
	return r.buf[0], nil
}

func NewSparse(data, offsets Appender) (*Sparse, error) {
	s := &Sparse{
		index:   C.sparse_create(),
		data:    data,
		offsets: offsets,
	}
	runtime.SetFinalizer(s, func(s *Sparse) {
		C.sparse_free(s.index)
	})
	offsetsSize, err := offsets.Size()
	if err != nil {
		return nil, err
	}
	offsetsReader := &byteReader{
		a:   offsets,
		n:   offsetsSize,
		buf: make([]byte, 1),
	}
	for {
		items := []uint64{0, 0, 0}
		for i := range items {
			items[i], err = binary.ReadUvarint(offsetsReader)
			if err == io.EOF && i == 0 {
				goto offsetsDone
			} else if err != nil {
				return nil, err
			}
		}
		off := C.int64_t(items[0])
		diskStart := C.int64_t(items[1])
		sliceLength := C.int64_t(items[2])
		C.sparse_write(s.index, off, diskStart, sliceLength)
	}
offsetsDone:
	return s, nil
}

func zero(buf []byte) {
	// https://stackoverflow.com/a/30614594
	if len(buf) == 0 {
		return
	}
	buf[0] = 0
	for bp := 1; bp < len(buf); bp *= 2 {
		copy(buf[bp:], buf[:bp])
	}
}

func (s *Sparse) ReadAt(p []byte, off int64) (int, error) {
	l := len(p)
	var sumn int
	for {
		var diskStart, sliceLength C.int64_t
		r := int64(C.sparse_read(s.index, C.int64_t(off), &diskStart, &sliceLength))
		if sliceLength != 0 {
			if int(sliceLength) > len(p) {
				sliceLength = C.int64_t(len(p))
			}
			n, err := s.data.ReadAt(p[:sliceLength], int64(diskStart))
			sumn += n
			if err != nil {
				return sumn, err
			} else if n != int(sliceLength) {
				return sumn, io.EOF
			}
			p = p[sliceLength:]
			off += int64(sliceLength)
			if len(p) == 0 {
				break
			}
		}
		if r == -1 || int(r) > len(p) {
			r = int64(len(p))
		}
		zero(p[:r])
		p = p[r:]
		sumn += int(r)
		off += int64(r)
		if len(p) == 0 {
			break
		}
	}
	return l, nil
}

var empty1024 = make([]byte, 1024)

func allZeros(buf []byte) bool {
	for len(buf) > 1024 {
		if !bytes.Equal(buf[:1024], empty1024) {
			return false
		}
		buf = buf[1024:]
	}
	if !bytes.Equal(buf, empty1024[:len(buf)]) {
		return false
	}
	return true
}

func (s *Sparse) WriteAt(p []byte, off int64) (int, error) {
	pn := len(p)
	// Locate this place.
	var diskStart0, sliceLength0 C.int64_t
	r := int64(C.sparse_read(s.index, C.int64_t(off), &diskStart0, &sliceLength0))
	if sliceLength0 == 0 && r == -1 || int(r) > len(p) {
		// Optimizations when writing to new place.
		if allZeros(p) {
			return pn, nil
		}
		for p[0] == 0 {
			p = p[1:]
			off++
		}
		for p[len(p)-1] == 0 {
			p = p[:len(p)-1]
		}
	}
	diskStart, err := s.data.Size()
	if err != nil {
		return 0, err
	}
	// Write to s.data.
	if n, err := s.data.Append(p); err != nil {
		return n, err
	} else if n != len(p) {
		return n, io.ErrShortWrite
	}
	// Write to s.offsets.
	items := []uint64{uint64(off), uint64(diskStart), uint64(len(p))}
	buf := make([]byte, binary.MaxVarintLen64)
	for _, item := range items {
		l := binary.PutUvarint(buf, item)
		if n, err := s.offsets.Append(buf[:l]); err != nil {
			return len(p), err
		} else if n != l {
			return len(p), io.ErrShortWrite
		}
	}
	C.sparse_write(s.index, C.int64_t(off), C.int64_t(diskStart), C.int64_t(len(p)))
	return pn, nil
}
