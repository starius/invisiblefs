package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/starius/invisiblefs/sparse"
	"github.com/starius/invisiblefs/sparse/sparsefuse"
)

var (
	mountpoint = flag.String("mountpoint", "", "Where to mount")
	dFile      = flag.String("data", ":memory:", "File with data")
	oFile      = flag.String("offsets", ":memory:", "File with offsets")
	virtFile   = flag.String("virt-file", "111", "Virtual file name")
	virtSize   = flag.Int64("virt-size", 100e9, "Virtual file size")
	mode       = flag.Int("mode", 1, "Mode (1 = only data file, 2 = both files")
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
	log.Printf("Total length: %d.", len(*a))
	return len(data), nil
}

func (a *DummyAppender) Size() (int64, error) {
	return int64(len(*a)), nil
}

type FileAppender struct {
	f *os.File
}

func (a *FileAppender) ReadAt(p []byte, off int64) (int, error) {
	return a.f.ReadAt(p, off)
}

func (a *FileAppender) Append(data []byte) (int, error) {
	return a.f.Write(data)
}

func (a *FileAppender) Size() (int64, error) {
	stat, err := a.f.Stat()
	if err != nil {
		return 0, err
	}
	return stat.Size(), nil
}

func open(name string) (sparse.Appender, func() error, error) {
	if name == ":memory:" {
		closer := func() error {
			return nil
		}
		return &DummyAppender{}, closer, nil
	} else {
		f, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
		if err != nil {
			return nil, nil, fmt.Errorf("OpenFile: %v", err)
		}
		fa := &FileAppender{f: f}
		closer := func() error {
			return f.Close()
		}
		return fa, closer, nil
	}
}

func main() {
	flag.Parse()
	if *mountpoint == "" {
		flag.PrintDefaults()
		log.Fatal("Provide -mountpoint")
	}
	data, dataCloser, err := open(*dFile)
	if err != nil {
		log.Fatal(err)
	}
	offsetsCloser := func() error {
		return nil
	}
	var s *sparse.Sparse
	if *mode == 2 {
		var offsets sparse.Appender
		offsets, offsetsCloser, err = open(*oFile)
		if err != nil {
			log.Fatal(err)
		}
		s, err = sparse.NewSparse2(data, offsets)
	} else if *mode == 1 {
		s, err = sparse.NewSparse1(data)
	} else {
		log.Fatalf("Unknown mode: %d.", *mode)
	}
	if err != nil {
		log.Fatalf("Failed to create sparse object: %s.", err)
	}
	f, err := sparsefuse.New(*virtFile, *virtSize, s)
	if err != nil {
		log.Fatalf("Failed to create fuse object: %s.", err)
	}
	sparsefuse.Run(*mountpoint, f, func() {
		fmt.Printf("Closing files.\n")
		if err := dataCloser(); err != nil {
			fmt.Printf("Failed to close data: %s.\n", err)
		} else {
			fmt.Printf("Successfully closed data.\n")
		}
		if err := offsetsCloser(); err != nil {
			fmt.Printf("Failed to close offsets: %s.\n", err)
		} else {
			fmt.Printf("Successfully closed offsets.\n")
		}
	})
}
