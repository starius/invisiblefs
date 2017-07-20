package crypto

import (
	"github.com/starius/invisiblefs/siaform/manager"
)

type SiaClient struct {
	c       *Cipher
	backend manager.SiaClient
}

func New(key []byte, backend manager.SiaClient) (*SiaClient, error) {
	c, err := NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &SiaClient{
		c:       c,
		backend: backend,
	}, err
}

func (s *SiaClient) Contracts() ([]string, error) {
	return s.backend.Contracts()
}

func (s *SiaClient) Read(contractID, sectorRoot string, sectorID int64) ([]byte, error) {
	data, err := s.backend.Read(contractID, sectorRoot, sectorID)
	if err != nil {
		return nil, err
	}
	data1 := make([]byte, len(data))
	copy(data1, data)
	s.c.Decrypt(sectorID, data1)
	return data1, err
}

func (s *SiaClient) Write(contractID string, data []byte, sectorID int64) (string, error) {
	data1 := make([]byte, len(data))
	copy(data1, data)
	s.c.Encrypt(sectorID, data1)
	return s.backend.Write(contractID, data1, sectorID)
}
