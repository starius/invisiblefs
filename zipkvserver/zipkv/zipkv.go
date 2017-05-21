package zipkv

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/golang/protobuf/proto"
)

//go:generate protoc --proto_path=. --go_out=. db.proto

const maxDbName = 9

type Header struct {
	Key      string
	Metadata []byte
	Size     int32
}

type KV interface {
	Has(key string) (bool, []byte, error)
	Get(key string) ([]byte, []byte, error)
	GetAt(key string, offset, size int) ([]byte, []byte, error)
	List() ([]Header, error)
	Put(key string, value, metadata []byte) error
	Link(dstKey, srcKey string, metadata []byte) error
	Delete(key string) (metadata []byte, err error)
	Sync() error
}

func Zip(backend KV, maxValueSize int) (*Frontend, error) {
	if maxValueSize <= 0 {
		return nil, fmt.Errorf("maxValueSize too small")
	}
	fe := &Frontend{
		be:  backend,
		max: maxValueSize,
	}
	if err := fe.setupDb(); err != nil {
		return nil, err
	}
	return fe, nil
}

type Frontend struct {
	be     KV
	max    int
	currDb int
	db     *Db
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

func (f *Frontend) setupDb() error {
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
		data, err := f.gunzip(zdata)
		if err != nil {
			return fmt.Errorf("f.gunzip(zdata): %s", err)
		}
		f.db = &Db{}
		if err := proto.Unmarshal(data, f.db); err != nil {
			return fmt.Errorf("proto.Unmarshal: %s", err)
		}
	} else {
		f.db = &Db{
			NextBackendFile: 0,
		}
	}
	if f.db.FrontendFiles == nil {
		f.db.FrontendFiles = make(map[string]*Location)
	}
	f.currDb = i
	return nil
}

func (f *Frontend) Has(key string) (bool, []byte, error) {
	f.m.RLock()
	defer f.m.RUnlock()
	loc, has := f.db.FrontendFiles[key]
	if has {
		return true, loc.Metadata, nil
	} else {
		return false, nil, nil
	}
}

func (f *Frontend) Get(key string) ([]byte, []byte, error) {
	f.m.RLock()
	loc, has := f.db.FrontendFiles[key]
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
	loc, has := f.db.FrontendFiles[key]
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
	data, _, err := f.be.GetAt(blockname, offset, size)
	return data, loc.Metadata, err
}

func (f *Frontend) List() ([]Header, error) {
	var list []Header
	for key, loc := range f.db.FrontendFiles {
		list = append(list, Header{
			Key:      key,
			Metadata: loc.Metadata,
			Size:     loc.Size,
		})
	}
	return list, nil
}

func (f *Frontend) writeDb() error {
	// Call this function under f.m.Lock().
	data, err := proto.Marshal(f.db)
	if err != nil {
		return fmt.Errorf("proto.Marshal(f.db): %s", err)
	}
	zdata, err := f.gzip(data)
	if err != nil {
		return fmt.Errorf("f.gzip(data): %s", err)
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
	f.next = nil
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
	if loc, has := f.db.FrontendFiles[key]; has {
		f.db.History = append(f.db.History, &HistoryRecord{
			Filename: key,
			Location: loc,
		})
	}
	f.db.FrontendFiles[key] = &Location{
		BackendFile: f.db.NextBackendFile,
		Offset:      int32(len(f.next)),
		Size:        int32(len(value)),
		Metadata:    metadata,
	}
	f.next = append(f.next, value...)
	return nil
}

func (f *Frontend) Link(dstKey, srcKey string, metadata []byte) error {
	f.m.Lock()
	defer f.m.Unlock()
	loc, has := f.db.FrontendFiles[srcKey]
	if !has {
		return fmt.Errorf("no key %q", srcKey)
	}
	prevLoc, has := f.db.FrontendFiles[dstKey]
	if has {
		f.db.History = append(f.db.History, &HistoryRecord{
			Filename: dstKey,
			Location: prevLoc,
		})
	}
	f.db.FrontendFiles[dstKey] = &Location{
		BackendFile: loc.BackendFile,
		Offset:      loc.Offset,
		Size:        loc.Size,
		Metadata:    metadata,
	}
	return nil
}

func (f *Frontend) Delete(key string) (metadata []byte, err error) {
	f.m.Lock()
	defer f.m.Unlock()
	loc, has := f.db.FrontendFiles[key]
	if !has {
		return loc.Metadata, fmt.Errorf("no key %q", key)
	}
	f.db.History = append(f.db.History, &HistoryRecord{
		Filename: key,
		Location: loc,
	})
	delete(f.db.FrontendFiles, key)
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

func (f *Frontend) gzip(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Flush(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (f *Frontend) gunzip(zdata []byte) ([]byte, error) {
	b := bytes.NewBuffer(zdata)
	gz, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(gz)
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return data, err
}
