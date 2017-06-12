package manager

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/starius/invisiblefs/gzip"
	"github.com/starius/invisiblefs/siaform/managerdb"
	"github.com/starius/invisiblefs/siaform/siaclient"
)

type Manager struct {
	db        *managerdb.Db
	siaclient *siaclient.SiaClient
	next      int64
	pending   []int64
	mu        sync.Mutex
}

func New(nprimary, necc int, sc *siaclient.SiaClient) (*Manager, error) {
	return &Manager{
		db: &managerdb.Db{
			PrimarySectors:        make(map[int64]*managerdb.Sector),
			EccSectors:            make(map[int64]*managerdb.Sector),
			PrimarySectorsInGroup: int32(nprimary),
			EccSectorsInGroup:     int32(necc),
		},
		siaclient: sc,
		next:      1,
	}, nil
}

func Load(zdump []byte, sc *siaclient.SiaClient) (*Manager, error) {
	dump, err := gzip.Gunzip(zdump)
	if err != nil {
		return nil, fmt.Errorf("gzip.Gunzip(zdump): %v", err)
	}
	db := &managerdb.Db{}
	if err := proto.Unmarshal(dump, db); err != nil {
		return nil, fmt.Errorf("proto.Unmarshal(dump, db): %v", err)
	}
	if db.PrimarySectors == nil {
		db.PrimarySectors = make(map[int64]*managerdb.Sector)
		db.EccSectors = make(map[int64]*managerdb.Sector)
	}
	next := int64(1)
	var pending []int64
	for i, sector := range db.PrimarySectors {
		if i > next {
			next = i
		}
		if sector.Data != nil {
			pending = append(pending, i)
		}
	}
	for i := range db.EccSectors {
		if i > next {
			next = i
		}
	}
	return &Manager{
		db:        db,
		siaclient: sc,
		next:      next,
		pending:   pending,
	}, nil
}

func (m *Manager) DumpDb() ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dump, err := proto.Marshal(m.db)
	if err != nil {
		return nil, fmt.Errorf("proto.Marshal: %v", err)
	}
	zdump, err := gzip.Gzip(dump)
	if err != nil {
		return nil, fmt.Errorf("gzip.Gzip: %v", err)
	}
	return zdump, nil
}

func (m *Manager) ReadSector(i int64) ([]byte, error) {
	m.mu.Lock()
	sector, has := m.db.PrimarySectors[i]
	m.mu.Unlock()
	if !has {
		return nil, fmt.Errorf("No such sector: %d", i)
	}
	if sector.Data != nil {
		return sector.Data, nil
	}
	contract := hex.EncodeToString(sector.Contract)
	sectorRoot := hex.EncodeToString(sector.MerkleRoot)
	return m.siaclient.Read(contract, sectorRoot)
}

func (m *Manager) ReadSectorAt(i int64, offset, length int) ([]byte, error) {
	// TODO: read only needed part.
	data, err := m.ReadSector(i)
	if err != nil {
		return nil, err
	}
	if offset < 0 || length < 0 || offset+length > len(data) {
		return nil, fmt.Errorf("Bad range: [%d,%d)", offset, offset+length)
	}
	return data[offset : offset+length], nil
}

func (m *Manager) AddSector(data []byte) (int64, error) {
	fmt.Printf("Manager.AddSector) start\n")
	defer fmt.Printf("Manager.AddSector) stop\n")
	m.mu.Lock()
	defer m.mu.Unlock()
	i := m.next
	m.next++
	m.db.PrimarySectors[i] = &managerdb.Sector{
		Data: data,
	}
	m.pending = append(m.pending, i)
	if int32(len(m.pending)) == m.db.PrimarySectorsInGroup {
		m.write(m.pending)
		m.pending = nil
	}
	return i, nil
}

func (m *Manager) write(pending []int64) {
	// The caller of this function holds m.mu locked.
	contracts, err := m.siaclient.Contracts()
	if err != nil {
		log.Printf("siaclient.Contracts: %v.", err)
		return
	}
	if len(contracts) == 0 {
		log.Printf("len(contracts) == 0")
		return
	}
	group := make(map[int64]*managerdb.Sector)
	for _, i := range pending {
		group[i] = &managerdb.Sector{
			Data: m.db.PrimarySectors[i].Data,
		}
	}
	for _, sector := range group {
		n := rand.Intn(len(contracts))
		contractID := contracts[n]
		sectorRoot, err := m.siaclient.Write(contractID, sector.Data)
		if err != nil {
			log.Printf("siaclient.Write: %v.", err)
			return
		}
		sector.Contract, err = hex.DecodeString(contractID)
		if err != nil {
			log.Printf("hex.DecodeString(%q): %v.", contractID, err)
			return
		}
		sector.MerkleRoot, err = hex.DecodeString(sectorRoot)
		if err != nil {
			log.Printf("hex.DecodeString(%q): %v.", sectorRoot, err)
			return
		}
		sector.Data = nil
	}
	for i, sector := range group {
		m.db.PrimarySectors[i] = sector
	}
	for i, sector := range m.db.PrimarySectors {
		fmt.Printf("%d - %d\n", i, len(sector.Data))
	}
}
