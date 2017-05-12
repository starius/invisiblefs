package devindir

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/starius/invisiblefs/inmem"
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
	m       sync.Mutex
	bs      int64
	size    int64
	dir     string
	bfcache *inmem.CloserCache
}

func New(dir, fname string, blockSize int, fileSize int64, filesCacheSize int) (*Fs, error) {
	stat, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("os.Stat(%q): %s", dir, err)
	}
	if !stat.Mode().IsDir() {
		return nil, fmt.Errorf("%q is not a dir", dir)
	}
	closer := func(f interface{}) {
		log.Printf("Closing %q", f.(*os.File).Name())
		if err := f.(*os.File).Close(); err != nil {
			log.Printf("Failed to close a file.")
		}
	}
	bfcache, err := inmem.NewCloserCache(filesCacheSize, closer)
	if err != nil {
		return nil, fmt.Errorf("inmem.NewCloserCache: %s", err)
	}
	return &Fs{
		d: &Dir{
			fname: fname,
			f: &File{
				bs:      int64(blockSize),
				size:    fileSize,
				dir:     dir,
				bfcache: bfcache,
			},
		},
	}, nil
}

/*
func (fs *Fs) Statfs(ctx context.Context, req *fuse.StatfsRequest, resp *fuse.StatfsResponse) error {
	resp.Bsize = 1024*1024
	return nil
}
*/

func (fs *Fs) Root() (fs.Node, error) {
	return fs.d, nil
}

func (fs *Fs) Close() error {
	return fs.d.f.Close()
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

func (f *File) blockName(b int64) string {
	name := fmt.Sprintf("block%010d", b)
	return filepath.Join(f.dir, name)
}

func (f *File) open(b int64, write bool) (*os.File, error) {
	f.m.Lock()
	defer f.m.Unlock()
	if bf, has := f.bfcache.Get(b); has {
		return bf.(*os.File), nil
	}
	fname := f.blockName(b)
	bf, err := os.OpenFile(fname, os.O_RDWR, 0600)
	if err == nil {
		f.bfcache.Add(b, bf)
		return bf, nil
	}
	if !write {
		return nil, err
	}
	bf, err = os.OpenFile(fname, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err == nil {
		f.bfcache.Add(b, bf)
	}
	return bf, err
}

func (f *File) blocks(offset, size int64) (int64, int64) {
	begin := offset / f.bs
	end := (offset+size-1)/f.bs + 1
	return begin, end
}

func (f *File) coords(b, offset, size int64) (int64, int64, int64) {
	destBegin := b*f.bs - offset
	destEnd := (b+1)*f.bs - offset
	fileBegin := int64(0)
	if destBegin < 0 {
		// First block, reading from the middle.
		fileBegin = -destBegin
		destBegin = 0
	}
	if destEnd > size {
		// Last block.
		destEnd = size
	}
	opSize := destEnd - destBegin
	return destBegin, opSize, fileBegin
}

func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, res *fuse.ReadResponse) error {
	//log.Printf("Read, off=%d size=%d", req.Offset, req.Size)
	if req.Offset >= f.size {
		res.Data = []byte{}
		return nil
	}
	res.Data = make([]byte, req.Size)
	begin, end := f.blocks(req.Offset, int64(req.Size))
	for b := begin; b < end; b++ {
		bf, err := f.open(b, false)
		if err != nil {
			// Probably reading from missing blocks.
			// TODO Can be other kind of error.
			continue
		}
		db, size, fb := f.coords(b, req.Offset, int64(req.Size))
		_, err = bf.ReadAt(res.Data[db:size], fb)
		if err != nil && err != io.EOF {
			return fuse.EIO
		}
		// If the file is less than f.bs, ReadAt does its job and
		// returns EOF.
	}
	return nil
}

func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, res *fuse.WriteResponse) error {
	//log.Printf("Write, off=%d size=%d", req.Offset, len(req.Data))
	if req.Offset >= f.size {
		log.Printf("%d > %d", req.Offset, f.size)
		return fuse.EIO
	}
	begin, end := f.blocks(req.Offset, int64(len(req.Data)))
	for b := begin; b < end; b++ {
		bf, err := f.open(b, true)
		if err != nil {
			log.Printf("open(%d): %s", b, err)
			return fuse.EIO
		}
		db, size, fb := f.coords(b, req.Offset, int64(len(req.Data)))
		n, err := bf.WriteAt(req.Data[db:size], fb)
		res.Size += n
		if err != nil && err != io.EOF {
			log.Printf("bf.WriteAt(req.Data[%d:%d], %d): %s", db, size, fb, err)
			return fuse.EIO
		}
	}
	return nil
}

func (f *File) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	f.m.Lock()
	defer f.m.Unlock()
	for _, entry := range f.bfcache.Items() {
		bf := entry.Value.(*os.File)
		if err := bf.Sync(); err != nil {
			log.Printf("Failed to sync file %q: %s.", bf.Name(), err)
			return fuse.EIO
		}
	}
	return nil
}

func (f *File) Close() error {
	f.m.Lock()
	defer f.m.Unlock()
	for _, entry := range f.bfcache.Items() {
		bf := entry.Value.(*os.File)
		log.Printf("Closing file %s.", bf.Name())
		if err := bf.Close(); err != nil {
			log.Printf("Failed to close file %q: %s.", bf.Name(), err)
		}
	}
	f.bfcache = nil // No open() must be called after this point.
	return nil
}
