package sparsefuse

import (
	"log"
	"os"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/starius/invisiblefs/sparse"
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
	s    *sparse.Sparse
	size int64
}

func New(fname string, size int64, s *sparse.Sparse) (*Fs, error) {
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
