package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/starius/invisiblefs/sparse"
	"github.com/starius/invisiblefs/sparse/sparsefuse"
)

var (
	mountpoint = flag.String("mountpoint", "", "Where to mount")
	dFile      = flag.String("data", ":memory:", "File with data")
	oFile      = flag.String("offsets", ":memory:", "File with offsets")
	virtFile   = flag.String("virt-file", "111", "Virtual file name")
	virtSize   = flag.Int64("virt-size", 100e9, "Virtual file size")
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

func open(name string) (sparse.Appender, error) {
	if name == ":memory:" {
		return &DummyAppender{}, nil
	} else {
		f, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
		if err != nil {
			return nil, fmt.Errorf("OpenFile: %v", err)
		}
		return &FileAppender{f: f}, nil
	}
}

func main() {
	flag.Parse()
	if *mountpoint == "" {
		flag.PrintDefaults()
		log.Fatal("Provide -mountpoint")
	}
	data, err := open(*dFile)
	if err != nil {
		log.Fatal(err)
	}
	offsets, err := open(*oFile)
	if err != nil {
		log.Fatal(err)
	}
	s, err := sparse.NewSparse(data, offsets)
	if err != nil {
		log.Fatalf("Failed to create sparse object: %s.", err)
	}
	f, err := sparsefuse.New(*virtFile, *virtSize, s)
	if err != nil {
		log.Fatalf("Failed to create fuse object: %s.", err)
	}
	mp, err := fuse.Mount(
		*mountpoint, fuse.FSName("sparse"),
		fuse.Subtype("sparse"), fuse.LocalVolume(),
		fuse.AllowOther(),
	)
	if err != nil {
		log.Fatalf("Failed to mount FUSE: %s.", err)
	}
	defer mp.Close()
	err = fs.Serve(mp, f)
	if err != nil {
		log.Fatal(err)
	}
	// check if the mount process has an error to report
	<-mp.Ready
	if err := mp.MountError; err != nil {
		log.Fatal(err)
	}
}
