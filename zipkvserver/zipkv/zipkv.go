package zipkv

import (
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/starius/invisiblefs/gzip"
	"github.com/starius/invisiblefs/zipkvserver/kv"
)

//go:generate protoc --proto_path=. --go_out=. db.proto

const maxDbName = 9

func Zip(backend kv.KV, maxValueSize int, rev int) (*Frontend, error) {
	if maxValueSize <= 0 {
		return nil, fmt.Errorf("maxValueSize too small")
	}
	fe := &Frontend{
		be:  backend,
		max: maxValueSize,
	}
	if err := fe.setupDb(rev); err != nil {
		return nil, err
	}
	return fe, nil
}

type Frontend struct {
	be     kv.KV
	max    int
	currDb int
	db     *Db
	files  map[string]*Location
	next   []byte
	m      sync.RWMutex
}

func (f *Frontend) dbName(i int) string {
	return fmt.Sprintf("db%010d", i)
}

func (f *Frontend) blockName(i int32) string {
	return fmt.Sprintf("block%010d", i)
}

func (f *Frontend) findDb() (int, error) {
	for i := 0; i <= maxDbName; i++ {
		dbname := f.dbName(i)
		if has, _, err := f.be.Has(dbname); err != nil {
			return 0, fmt.Errorf("f.be.Has(%q): %s", dbname, err)
		} else if has {
			return i, nil
		}
	}
	return -1, nil
}

func (f *Frontend) setupDb(rev int) error {
	f.files = make(map[string]*Location)
	i, err := f.findDb()
	if err != nil {
		return err
	}
	if i != -1 {
		dbname := f.dbName(i)
		zdata, _, err := f.be.Get(dbname)
		if err != nil {
			return fmt.Errorf("f.be.Get(%q): %s", dbname, err)
		}
		data, err := gzip.Gunzip(zdata)
		if err != nil {
			return fmt.Errorf("gzip.Gunzip(zdata): %s", err)
		}
		f.db = &Db{}
		if err := proto.Unmarshal(data, f.db); err != nil {
			return fmt.Errorf("proto.Unmarshal: %s", err)
		}
		if rev != -1 {
			wantLen := rev + 1
			if wantLen > len(f.db.History) {
				return fmt.Errorf("rev (%d) is too high", rev)
			}
			f.db.History = f.db.History[:wantLen]
		}
		// Fill f.files based on the f.db.History.
		for _, record := range f.db.History {
			switch r := record.Record.(type) {
			default:
				panic(fmt.Sprintf("record of type %T", r))
			case *HistoryRecord_Put:
				f.files[r.Put.Filename] = r.Put.Location
			case *HistoryRecord_Delete:
				delete(f.files, r.Delete.Filename)
			}
		}
	} else {
		f.db = &Db{
			NextBackendFile: 0,
		}
	}
	f.currDb = i
	return nil
}

func (f *Frontend) Has(key string) (bool, []byte, error) {
	f.m.RLock()
	defer f.m.RUnlock()
	loc, has := f.files[key]
	if has {
		return true, loc.Metadata, nil
	} else {
		return false, nil, nil
	}
}

func (f *Frontend) Get(key string) ([]byte, []byte, error) {
	f.m.RLock()
	loc, has := f.files[key]
	if !has {
		f.m.RUnlock()
		return nil, nil, fmt.Errorf("no key %q", key)
	}
	if loc.BackendFile == f.db.NextBackendFile {
		data := f.next[loc.Offset : loc.Offset+loc.Size]
		f.m.RUnlock()
		return data, loc.Metadata, nil
	}
	f.m.RUnlock()
	blockname := f.blockName(loc.BackendFile)
	data, _, err := f.be.GetAt(blockname, int(loc.Offset), int(loc.Size))
	return data, loc.Metadata, err
}

func (f *Frontend) GetAt(key string, offset, size int) ([]byte, []byte, error) {
	f.m.RLock()
	loc, has := f.files[key]
	if !has {
		f.m.RUnlock()
		return nil, nil, fmt.Errorf("no key %q", key)
	}
	if offset < 0 {
		f.m.RUnlock()
		return nil, loc.Metadata, fmt.Errorf("offset < 0")
	}
	if offset+size > int(loc.Size) {
		f.m.RUnlock()
		return nil, loc.Metadata, fmt.Errorf("%d+%d > %d", offset, size, loc.Size)
	}
	offset2 := int(loc.Offset) + offset
	if loc.BackendFile == f.db.NextBackendFile {
		data := f.next[offset2 : offset2+size]
		f.m.RUnlock()
		return data, loc.Metadata, nil
	}
	f.m.RUnlock()
	blockname := f.blockName(loc.BackendFile)
	data, _, err := f.be.GetAt(blockname, offset2, size)
	return data, loc.Metadata, err
}

func (f *Frontend) List() (map[string]int, error) {
	sizes := make(map[string]int)
	for key, loc := range f.files {
		sizes[key] = int(loc.Size)
	}
	return sizes, nil
}

func (f *Frontend) writeDb() error {
	// Call this function under f.m.Lock().
	data, err := proto.Marshal(f.db)
	if err != nil {
		return fmt.Errorf("proto.Marshal(f.db): %s", err)
	}
	zdata, err := gzip.Gzip(data)
	if err != nil {
		return fmt.Errorf("gzip.Gzip(data): %s", err)
	}
	nextDb := (f.currDb + 1) % (maxDbName + 1)
	dbname := f.dbName(nextDb)
	if err := f.be.Put(dbname, zdata, nil); err != nil {
		return fmt.Errorf("f.be.Put(%q, ...): %s", dbname, err)
	}
	if f.currDb != -1 {
		prevname := f.dbName(f.currDb)
		if _, err := f.be.Delete(prevname); err != nil {
			return fmt.Errorf("f.be.Delete(%q): %s", prevname, err)
		}
	}
	f.currDb = nextDb
	return nil
}

func (f *Frontend) writeNext() error {
	// Call this function under f.m.Lock().
	blockname := f.blockName(f.db.NextBackendFile)
	if err := f.be.Put(blockname, f.next, nil); err != nil {
		return fmt.Errorf("f.be.Put(%q, ...): %s", blockname, err)
	}
	f.db.NextBackendFile++
	if err := f.writeDb(); err != nil {
		return fmt.Errorf("f.writeDb(): %s", err)
	}
	f.next = f.next[:0]
	return nil
}

func (f *Frontend) Put(key string, value, metadata []byte) error {
	if len(value) > f.max {
		return fmt.Errorf("%d > %d", len(value), f.max)
	}
	f.m.Lock()
	defer f.m.Unlock()
	if len(f.next)+len(value) > f.max {
		if err := f.writeNext(); err != nil {
			return fmt.Errorf("f.writeNext(): %s", err)
		}
	}
	loc := &Location{
		BackendFile: f.db.NextBackendFile,
		Offset:      int32(len(f.next)),
		Size:        int32(len(value)),
		Metadata:    metadata,
	}
	f.files[key] = loc
	f.db.History = append(f.db.History, &HistoryRecord{
		Record: &HistoryRecord_Put{
			&PutRecord{
				Filename: key,
				Location: loc,
			},
		},
	})
	f.next = append(f.next, value...)
	return nil
}

func (f *Frontend) Link(dstKey, srcKey string, metadata []byte) error {
	f.m.Lock()
	defer f.m.Unlock()
	loc, has := f.files[srcKey]
	if !has {
		return fmt.Errorf("no key %q", srcKey)
	}
	newLoc := &Location{}
	*newLoc = *loc
	newLoc.Metadata = metadata
	f.files[dstKey] = newLoc
	f.db.History = append(f.db.History, &HistoryRecord{
		Record: &HistoryRecord_Put{
			&PutRecord{
				Filename: dstKey,
				Location: newLoc,
			},
		},
	})
	return nil
}

func (f *Frontend) Delete(key string) (metadata []byte, err error) {
	f.m.Lock()
	defer f.m.Unlock()
	loc, has := f.files[key]
	if !has {
		return nil, fmt.Errorf("no key %q", key)
	}
	f.db.History = append(f.db.History, &HistoryRecord{
		Record: &HistoryRecord_Delete{
			&DeleteRecord{
				Filename: key,
			},
		},
	})
	delete(f.files, key)
	return loc.Metadata, nil
}

func (f *Frontend) Sync() error {
	f.m.Lock()
	defer f.m.Unlock()
	if len(f.next) > 0 {
		if err := f.writeNext(); err != nil {
			return fmt.Errorf("f.writeNext(): %s", err)
		}
	}
	return nil
}

type Change struct {
	Put      bool // Otherwise Delete.
	Filename string
}

func (f *Frontend) History() []Change {
	history := make([]Change, len(f.db.History))
	for i, record := range f.db.History {
		var change Change
		switch r := record.Record.(type) {
		default:
			panic(fmt.Sprintf("record of type %T", r))
		case *HistoryRecord_Put:
			change.Put = true
			change.Filename = r.Put.Filename
		case *HistoryRecord_Delete:
			change.Put = false
			change.Filename = r.Delete.Filename
		}
		history[i] = change
	}
	return history
}
