package files

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/starius/invisiblefs/gzip"
	"github.com/starius/invisiblefs/siaform/filesdb"
	"github.com/starius/invisiblefs/siaform/manager"
)

type Files struct {
	db         *filesdb.Db
	manager    *manager.Manager
	sectorSize int
	mu         sync.Mutex
}

func New(sectorSize int, manager *manager.Manager) (*Files, error) {
	return &Files{
		db: &filesdb.Db{
			Files:      make(map[string]*filesdb.File),
			SectorSize: int32(sectorSize),
		},
		manager: manager,
	}, nil
}

func Load(zdump []byte, manager *manager.Manager) (*Files, error) {
	dump, err := gzip.Gunzip(zdump)
	if err != nil {
		return nil, fmt.Errorf("gzip.Gunzip(zdump): %v", err)
	}
	db := &filesdb.Db{}
	if err := proto.Unmarshal(dump, db); err != nil {
		return nil, fmt.Errorf("proto.Unmarshal(dump, db): %v", err)
	}
	if db.Files == nil {
		db.Files = make(map[string]*filesdb.File)
	}
	return &Files{
		db:      db,
		manager: manager,
	}, nil
}

func (f *Files) DumpDb() ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	dump, err := proto.Marshal(f.db)
	if err != nil {
		return nil, fmt.Errorf("proto.Marshal: %v", err)
	}
	zdump, err := gzip.Gzip(dump)
	if err != nil {
		return nil, fmt.Errorf("gzip.Gzip: %v", err)
	}
	return zdump, nil
}

func (f *Files) Open(name string) (*File, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f1, ok := f.db.Files[name]
	if !ok {
		return nil, fmt.Errorf("No file %q", name)
	}
	return &File{
		offset:       0,
		File:         f1,
		manager:      f.manager,
		sectorSize:   int(f.db.SectorSize),
		lastSectorID: -1,
	}, nil
}

func (f *Files) Create(name string) (*File, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.db.Files[name]
	if ok {
		return nil, fmt.Errorf("File exists: %q", name)
	}
	f1 := &filesdb.File{}
	f.db.Files[name] = f1
	return &File{
		offset:       0,
		File:         f1,
		manager:      f.manager,
		sectorSize:   int(f.db.SectorSize),
		lastSectorID: -1,
	}, nil
}

func (f *Files) Rename(oldName, newName string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f1, ok := f.db.Files[oldName]
	if !ok {
		return fmt.Errorf("No such file: %q", oldName)
	}
	f.db.Files[newName] = f1
	delete(f.db.Files, oldName)
	return nil
}

type File struct {
	offset       int64
	File         *filesdb.File
	manager      *manager.Manager
	sectorSize   int
	lastSector   []byte
	lastSectorID int64
	mu           sync.Mutex
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	log.Printf("Seek(%v, %v)\n", offset, whence)
	f.mu.Lock()
	defer f.mu.Unlock()
	if offset < 0 {
		return f.offset, fmt.Errorf("negative offset")
	}
	if whence == io.SeekStart {
		f.offset = offset
	} else if whence == io.SeekCurrent {
		f.offset += offset
	} else if whence == io.SeekEnd {
		f.offset = f.File.Size + offset
	} else {
		return f.offset, fmt.Errorf("unknown whence: %d", whence)
	}
	return f.offset, nil
}

func max(x, y int64) int64 {
	if x > y {
		return x
	} else {
		return y
	}
}

func min(x, y int64) int64 {
	if x > y {
		return y
	} else {
		return x
	}
}

func (f *File) Read(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fbegin := int64(0)
	pbegin := f.offset
	pend := pbegin + int64(len(p))
	for _, piece := range f.File.Pieces {
		offset := int64(0)
		length := int64(f.sectorSize)
		if len(piece.Sha256) != 0 {
			offset = int64(piece.Offset)
			length = int64(piece.Length)
		}
		fend := fbegin + length
		if (pbegin <= fbegin && fbegin < pend) || (fbegin <= pbegin && pbegin < fend) {
			begin := max(pbegin, fbegin)
			end := min(pend, fend)
			l := end - begin
			rbegin := begin - pbegin
			rend := rbegin + l
			sbegin := begin - fbegin + offset
			send := sbegin + l
			r := p[rbegin:rend]
			//
			sectorID := piece.SectorId
			var part []byte
			if len(piece.Sha256) == 0 {
				var sector []byte
				if sectorID == f.lastSectorID {
					sector = f.lastSector
				} else {
					sector, err = f.manager.ReadSector(sectorID)
					if err != nil {
						return n, err
					}
					f.lastSector = sector
					f.lastSectorID = sectorID
				}
				part = sector[sbegin:send]
			} else {
				part, err = f.manager.InsecureReadSectorAt(sectorID, int(sbegin), int(send))
				if err != nil {
					return n, err
				}
				checksum := sha256.Sum256(part)
				if !bytes.Equal(checksum[:], piece.Sha256) {
					return n, fmt.Errorf("Checksum mismatch")
				}
			}
			nn := copy(r, part)
			n += nn
			f.offset += int64(nn)
		}
		fbegin = fend
	}
	return n, nil
}

func (f *File) Write(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	log.Printf("Writing %d bytes.\n", len(p))
	if f.File.Size%int64(f.sectorSize) != 0 {
		return 0, fmt.Errorf("last block was the last one")
	}
	if len(p) > f.sectorSize {
		return 0, fmt.Errorf("too long write")
	}
	l := len(p)
	if l != f.sectorSize {
		zeros := make([]byte, f.sectorSize-l)
		p = append(p, zeros...)
	}
	sectorID, err := f.manager.AddSector(p)
	if err != nil {
		return 0, fmt.Errorf("AddSector: %v", err)
	}
	f.File.Pieces = append(f.File.Pieces, &filesdb.Piece{
		SectorId: sectorID,
	})
	f.File.Size += int64(l)
	f.offset += int64(l)
	return l, nil
}
