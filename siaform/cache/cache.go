package cache

import (
	"sync"

	"github.com/starius/invisiblefs/inmem"
	"github.com/starius/invisiblefs/siaform/manager"
)

type SiaClient struct {
	backend manager.SiaClient
	cache   *inmem.CloserCache // Closer feature is not used.

	// To serialize requests to the same sector.
	inFlight map[string]struct{}
	mu       sync.Mutex
	cond     *sync.Cond
}

func New(size int, backend manager.SiaClient) (*SiaClient, error) {
	cache, err := inmem.NewCloserCache(size, func(interface{}) {})
	if err != nil {
		return nil, err
	}
	sc := &SiaClient{
		backend:  backend,
		cache:    cache,
		inFlight: make(map[string]struct{}),
	}
	sc.cond = sync.NewCond(&sc.mu)
	return sc, nil
}

func (s *SiaClient) Contracts() ([]string, error) {
	return s.backend.Contracts()
}

func (s *SiaClient) serialize(sectorRoot string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		if _, has := s.inFlight[sectorRoot]; !has {
			break
		}
		s.cond.Wait()
	}
	s.inFlight[sectorRoot] = struct{}{}
}

func (s *SiaClient) Read(contractID, sectorRoot string, sectorID int64) ([]byte, error) {
	s.serialize(sectorRoot)
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.inFlight, sectorRoot)
		s.cond.Broadcast()
	}()
	cached, has := s.cache.Get(sectorRoot)
	if has {
		return cached.([]byte), nil
	}
	data, err := s.backend.Read(contractID, sectorRoot, sectorID)
	if err == nil {
		s.cache.Add(sectorRoot, data)
	}
	return data, err
}

func (s *SiaClient) Write(contractID string, data []byte, sectorID int64) (string, error) {
	sectorRoot, err := s.backend.Write(contractID, data, sectorID)
	if err == nil {
		s.cache.Add(sectorRoot, data)
	}
	return sectorRoot, err
}
