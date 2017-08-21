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
	diskStart     int64

	prevOff, prevDiskStart, prevSliceLength int64
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
		offDiff, err := binary.ReadVarint(offsetsReader)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		diskStartDiff, err := binary.ReadUvarint(offsetsReader)
		if err != nil {
			return nil, err
		}
		sliceLengthDiff, err := binary.ReadVarint(offsetsReader)
		if err != nil {
			return nil, err
		}
		off := s.prevOff + offDiff
		diskStart := s.prevDiskStart + int64(diskStartDiff)
		sliceLength := s.prevSliceLength + sliceLengthDiff
		C.sparse_write(s.index, C.int64_t(off), C.int64_t(diskStart), C.int64_t(sliceLength))
		s.prevOff = off
		s.prevDiskStart = diskStart
		s.prevSliceLength = sliceLength
	}
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
	if s.diskStart == 0 {
		var err error
		s.diskStart, err = s.data.Size()
		if err != nil {
			s.diskStart = 0
			return 0, err
		}
	}
	// Write to s.data.
	if n, err := s.data.Append(p); err != nil {
		s.diskStart = 0
		return n, err
	} else if n != len(p) {
		s.diskStart = 0
		return n, io.ErrShortWrite
	}
	// Write to s.offsets.
	sliceLength := int64(len(p))
	offDiff := off - s.prevOff
	diskStartDiff := uint64(s.diskStart - s.prevDiskStart)
	sliceLengthDiff := sliceLength - s.prevSliceLength
	buf := make([]byte, 3*binary.MaxVarintLen64)
	buf1 := buf
	l := binary.PutVarint(buf1, offDiff)
	buf1 = buf1[l:]
	l = binary.PutUvarint(buf1, diskStartDiff)
	buf1 = buf1[l:]
	l = binary.PutVarint(buf1, sliceLengthDiff)
	buf1 = buf1[l:]
	buf = buf[:len(buf)-len(buf1)]
	if n, err := s.offsets.Append(buf); err != nil {
		s.diskStart = 0
		return len(p), err
	} else if n != len(buf) {
		s.diskStart = 0
		return len(p), io.ErrShortWrite
	}
	s.prevOff = off
	s.prevDiskStart = s.diskStart
	s.prevSliceLength = sliceLength
	C.sparse_write(s.index, C.int64_t(off), C.int64_t(s.diskStart), C.int64_t(sliceLength))
	s.diskStart += int64(len(p))
	return pn, nil
}
