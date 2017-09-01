package sparse

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"runtime"
	"sync"
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

// In case just one file is used, structure of one entry in the data file:
//   data_len = uvarint(len(data))
//   data
//   parent_ptr = uvarint(&(this.parent_ptr) - &(parent.parent_ptr) OR 0)
//   offsets_len = uvarint(len(offsets))
//   offsets, 3n varints
//   tail = 32 bit little endian of &(tail) - &(parent_ptr)

type Sparse struct {
	index         *C.Index
	data, offsets Appender // Field offsets is not used in one file case.
	dataSize      int64
	broken        bool

	mu sync.RWMutex

	prevOff, prevDiskStart, prevSliceLength int64

	// One file case.
	offsetsBytes []byte
	c            *chain
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

func readAt(r io.ReaderAt, off, length int64) ([]byte, error) {
	data := make([]byte, length)
	if n, err := r.ReadAt(data, off); err != nil {
		return nil, err
	} else if int64(n) != length {
		return nil, fmt.Errorf("short read")
	}
	return data, nil
}

func (s *Sparse) putOffsets(o []byte) (int, error) {
	nRecords := 0
	for len(o) > 0 {
		offDiff, n := binary.Varint(o)
		if n <= 0 {
			return 0, fmt.Errorf("reading offsets: bad varint")
		}
		o = o[n:]
		diskStartDiff, n := binary.Uvarint(o)
		if n <= 0 {
			return 0, fmt.Errorf("reading offsets: bad varint")
		}
		o = o[n:]
		sliceLengthDiff, n := binary.Varint(o)
		if n <= 0 {
			return 0, fmt.Errorf("reading offsets: bad varint")
		}
		o = o[n:]
		off := s.prevOff + offDiff
		diskStart := s.prevDiskStart + int64(diskStartDiff)
		sliceLength := s.prevSliceLength + sliceLengthDiff
		C.sparse_write(s.index, C.int64_t(off), C.int64_t(diskStart), C.int64_t(sliceLength))
		s.prevOff = off
		s.prevDiskStart = diskStart
		s.prevSliceLength = sliceLength
		nRecords++
	}
	return nRecords, nil
}

func NewSparse2(data, offsets Appender) (*Sparse, error) {
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
	s.dataSize, err = s.data.Size()
	if err != nil {
		return nil, err
	}
	if s.dataSize != s.prevDiskStart+s.prevSliceLength {
		return nil, fmt.Errorf("data size doesn't match records offsets: %d != %d",
			s.dataSize, s.prevDiskStart+s.prevSliceLength,
		)
	}
	return s, nil
}

func NewSparse1(data Appender) (*Sparse, error) {
	s := &Sparse{
		index: C.sparse_create(),
		data:  data,
	}
	runtime.SetFinalizer(s, func(s *Sparse) {
		C.sparse_free(s.index)
	})
	var err error
	s.dataSize, err = data.Size()
	if err != nil {
		return nil, fmt.Errorf("data.Size: %v", err)
	}
	if s.dataSize == 0 {
		s.c = newChain()
		return s, nil
	}
	// Build offsetsList by reading tail and parentPtr.
	if s.dataSize < 4 {
		return nil, fmt.Errorf("Too short tail")
	}
	tailBytes, err := readAt(s.data, s.dataSize-4, 4)
	if err != nil {
		return nil, fmt.Errorf("reading tail: %v", err)
	}
	tail := binary.LittleEndian.Uint32(tailBytes)
	if tail < 5 {
		return nil, fmt.Errorf("too small tail")
	}
	parentPtr := s.dataSize - 4 - int64(tail)
	var offsetsList [][]byte
	var ptrList []int64
	for {
		if parentPtr < 0 {
			return nil, fmt.Errorf("negative parentPtr")
		}
		header, err := readAt(s.data, parentPtr, s.dataSize-4-parentPtr)
		if err != nil {
			return nil, fmt.Errorf("reading header: %v", err)
		}
		parentPtrDiff, n := binary.Uvarint(header)
		if n <= 0 {
			return nil, fmt.Errorf("reading parentPtrDiff: bad varint")
		}
		header = header[n:]
		offsetsSize, n := binary.Uvarint(header)
		if n <= 0 {
			return nil, fmt.Errorf("reading offsetsSize: bad varint")
		}
		header = header[n:]
		if int(offsetsSize) > len(header) {
			return nil, fmt.Errorf("offsets overflow")
		}
		offsetsList = append(offsetsList, header[:offsetsSize])
		ptrList = append(ptrList, parentPtr)
		if parentPtrDiff == 0 {
			break
		}
		parentPtr -= int64(parentPtrDiff)
	}
	// Build and parse s.offsetsBytes.
	var elements []chainElem
	var sizes []int
	for j := len(offsetsList) - 1; j >= 0; j-- {
		o := offsetsList[j]
		s.offsetsBytes = append(s.offsetsBytes, o...)
		elem := chainElem{
			inOffsets: int64(len(s.offsetsBytes)),
			inData:    ptrList[j],
		}
		elements = append(elements, elem)
		nRecords, err := s.putOffsets(o)
		if err != nil {
			return nil, err
		}
		sizes = append(sizes, nRecords)
	}
	s.c, err = restore(elements, sizes)
	if err != nil {
		return nil, err
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
	s.mu.RLock()
	defer s.mu.RUnlock()
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.broken {
		return 0, fmt.Errorf("The storage was spoiled in a previous operation")
	}
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
	var n int
	var err error
	if s.offsets != nil {
		n, err = s.writeAt2(p, off)
	} else {
		n, err = s.writeAt1(p, off)
	}
	if err != nil {
		return n, err
	} else {
		return pn, nil
	}
}

func (s *Sparse) writeOffsets(off, diskStart, sliceLength int64) []byte {
	offDiff := off - s.prevOff
	diskStartDiff := uint64(diskStart - s.prevDiskStart)
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
	s.prevOff = off
	s.prevDiskStart = diskStart
	s.prevSliceLength = sliceLength
	return buf
}

func (s *Sparse) writeAt2(p []byte, off int64) (int, error) {
	// Write to s.data.
	if n, err := s.data.Append(p); err != nil {
		s.broken = true
		return n, err
	} else if n != len(p) {
		s.broken = true
		return n, io.ErrShortWrite
	}
	// Write to s.offsets.
	diskStart := s.dataSize
	sliceLength := int64(len(p))
	buf := s.writeOffsets(off, diskStart, sliceLength)
	if n, err := s.offsets.Append(buf); err != nil {
		s.broken = true
		return len(p), err
	} else if n != len(buf) {
		s.broken = true
		return len(p), io.ErrShortWrite
	}
	C.sparse_write(s.index, C.int64_t(off), C.int64_t(diskStart), C.int64_t(sliceLength))
	s.dataSize += sliceLength
	return len(p), nil
}

func (s *Sparse) writeAt1(p []byte, off int64) (int, error) {
	// Prepare entry in memory.
	parent := s.c.parent()
	offsetsSizeEstimate := len(s.offsetsBytes) - int(parent.inOffsets)
	buf := make([]byte, len(p)+offsetsSizeEstimate+6*binary.MaxVarintLen64+4)
	buf1 := buf
	n := binary.PutUvarint(buf1, uint64(len(p)))
	buf1 = buf1[n:]
	dataStart := s.dataSize + int64(len(buf)-len(buf1))
	copy(buf1, p)
	buf1 = buf1[len(p):]
	// Write parentPtr.
	parentPtrStart := s.dataSize + int64(len(buf)-len(buf1))
	var parentPtrDiff uint64 = 0
	if parent.inData != 0 {
		parentPtrDiff = uint64(parentPtrStart - parent.inData)
	}
	n = binary.PutUvarint(buf1, parentPtrDiff)
	buf1 = buf1[n:]
	// Write to s.offsetsBytes.
	offsetsBuf := s.writeOffsets(off, dataStart, int64(len(p)))
	s.offsetsBytes = append(s.offsetsBytes, offsetsBuf...)
	// Write len(offsetsBytes) and offsetsBytes to buf1.
	offsets := s.offsetsBytes[parent.inOffsets:]
	n = binary.PutUvarint(buf1, uint64(len(offsets)))
	buf1 = buf1[n:]
	copy(buf1, offsets)
	buf1 = buf1[len(offsets):]
	// Write tail.
	tailPtr := s.dataSize + int64(len(buf)-len(buf1))
	tail := uint32(tailPtr - parentPtrStart)
	binary.LittleEndian.PutUint32(buf1, tail)
	buf1 = buf1[4:]
	buf = buf[:len(buf)-len(buf1)]
	// Write buf.
	if n, err := s.data.Append(buf); err != nil {
		s.broken = true
		return n, err
	} else if n != len(buf) {
		s.broken = true
		return n, io.ErrShortWrite
	}
	s.dataSize += int64(len(buf))
	// Update s.offsetsBytes.
	s.c.push(chainElem{
		inOffsets: int64(len(s.offsetsBytes)),
		inData:    parentPtrStart,
	})
	// Update s.index.
	C.sparse_write(s.index, C.int64_t(off), C.int64_t(dataStart), C.int64_t(len(p)))
	return len(p), nil
}
