package kv

type KV interface {
	Has(key string) (bool, []byte, error)
	Get(key string) ([]byte, []byte, error)
	GetAt(key string, offset, size int) ([]byte, []byte, error)
	List() (map[string]int, error)
	Put(key string, value, metadata []byte) error
	Link(dstKey, srcKey string, metadata []byte) error
	Delete(key string) (metadata []byte, err error)
	Sync() error
}
