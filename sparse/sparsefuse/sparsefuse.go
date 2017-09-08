package sparsefuse

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type Fs struct {
	d *Dir
}

type Dir struct {
	fname string
	f     *File
}

type File struct {
	m    sync.Mutex
	s    file
	size int64
}

type file interface {
	io.ReaderAt
	io.WriterAt
}

func New(fname string, size int64, s file) (*Fs, error) {
	f := &File{
		s:    s,
		size: size,
	}
	d := &Dir{
		fname: fname,
		f:     f,
	}
	fs := &Fs{
		d: d,
	}
	return fs, nil
}

func (fs *Fs) Root() (fs.Node, error) {
	return fs.d, nil
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return []fuse.Dirent{
		{
			Name: d.fname,
			Type: fuse.DT_File,
		},
	}, nil
}

func (d *Dir) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Inode = 1
	attr.Mode = os.ModeDir | 0500
	attr.Uid = uint32(os.Getuid())
	attr.Gid = uint32(os.Getgid())
	return nil
}

func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if name != d.fname {
		return nil, fuse.ENOENT
	}
	return d.f, nil
}

func (f *File) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Inode = 2
	attr.Mode = 0600
	attr.Uid = uint32(os.Getuid())
	attr.Gid = uint32(os.Getgid())
	attr.Size = uint64(f.size)
	// TODO attr.Blocks
	return nil
}

func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, res *fuse.ReadResponse) error {
	log.Printf("Read, off=%d size=%d", req.Offset, req.Size)
	res.Data = make([]byte, req.Size)
	n, err := f.s.ReadAt(res.Data, req.Offset)
	if err != nil || n != len(res.Data) {
		return fuse.EIO
	}
	return nil
}

func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, res *fuse.WriteResponse) error {
	log.Printf("Write, off=%d size=%d", req.Offset, len(req.Data))
	n, err := f.s.WriteAt(req.Data, req.Offset)
	if err != nil || n != len(req.Data) {
		log.Printf("Write: err=%v, n=%d", err, n)
		return fuse.EIO
	}
	res.Size = n
	return nil
}

func (f *File) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	return nil
}

func Run(mountpoint string, f fs.FS, closer func()) {
	mp, err := fuse.Mount(
		mountpoint, fuse.FSName("sparse"),
		fuse.Subtype("sparse"), fuse.LocalVolume(),
		fuse.AllowOther(),
	)
	if err != nil {
		log.Fatalf("Failed to mount FUSE: %s.", err)
	}
	defer mp.Close()
	var wg sync.WaitGroup
	// Handle signals.
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	wg.Add(1)
	go func() {
		defer wg.Done()
		sig := <-c
		fmt.Printf("Caught %s.\n", sig)
		fmt.Printf("Unmounting %s.\n", mountpoint)
		if err := fuse.Unmount(mountpoint); err != nil {
			fmt.Printf("Failed to unmount: %s.\n", err)
		} else {
			fmt.Printf("Successfully unmount %s.\n", mountpoint)
		}
		closer()
		fmt.Printf("Exiting.\n")
	}()
	if err = fs.Serve(mp, f); err != nil {
		log.Fatal(err)
	}
	// Check if the mount process has an error to report.
	<-mp.Ready
	if err := mp.MountError; err != nil {
		log.Fatal(err)
	}
	wg.Wait()
}
