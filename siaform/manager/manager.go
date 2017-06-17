package manager

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/klauspost/reedsolomon"
	"github.com/starius/invisiblefs/gzip"
	"github.com/starius/invisiblefs/siaform/managerdb"
	"github.com/starius/invisiblefs/siaform/siaclient"
)

func hex2bytes(data string) []byte {
	bytes, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}
	return bytes
}

func bytes2hex(data []byte) string {
	return hex.EncodeToString(data)
}

type Sector struct {
	Contract   string
	MerkleRoot string
	Data       []byte

	isPrimary bool
	set       *Set
	id        int64
}

type Set struct {
	DataSectors, ParitySectors []*Sector
}

type Manager struct {
	sectors map[int64]*Sector
	sets    []*Set
	next    int64
	pending []*Sector
	mu      sync.Mutex

	siaclient *siaclient.SiaClient

	ndata, nparity, sectorSize int

	dataChan chan *Sector
	setChan  chan *Set
	stopChan chan struct{}
}

func New(ndata, nparity, sectorSize int, sc *siaclient.SiaClient) (*Manager, error) {
	return &Manager{
		sectors:    make(map[int64]*Sector),
		next:       1,
		ndata:      ndata,
		nparity:    nparity,
		sectorSize: sectorSize,
		siaclient:  sc,
		dataChan:   make(chan *Sector, 100),
		setChan:    make(chan *Set, 100),
		stopChan:   make(chan struct{}),
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
	m := &Manager{
		sectors:    make(map[int64]*Sector),
		siaclient:  sc,
		dataChan:   make(chan *Sector, 100),
		setChan:    make(chan *Set, 100),
		stopChan:   make(chan struct{}),
		ndata:      int(db.Ndata),
		nparity:    int(db.Nparity),
		sectorSize: int(db.SectorSize),
	}
	maxI := int64(0)
	for i, sector := range db.Sectors {
		if i > maxI {
			maxI = i
		}
		m.sectors[i] = &Sector{
			Contract:   bytes2hex(sector.Contract),
			MerkleRoot: bytes2hex(sector.MerkleRoot),
			Data:       sector.Data,
			id:         i,
		}
	}
	m.next = maxI + 1
	for _, set := range db.Sets {
		set1 := &Set{}
		for _, i := range set.DataIds {
			sector := m.sectors[i]
			set1.DataSectors = append(set1.DataSectors, sector)
			sector.set = set1
			sector.isPrimary = true
		}
		for _, i := range set.ParityIds {
			sector := m.sectors[i]
			set1.ParitySectors = append(set1.ParitySectors, sector)
			sector.set = set1
		}
	}
	return m, nil
}

func (m *Manager) DumpDb() ([]byte, error) {
	db := &managerdb.Db{
		Sectors:    make(map[int64]*managerdb.Sector),
		Ndata:      int32(m.ndata),
		Nparity:    int32(m.nparity),
		SectorSize: int32(m.sectorSize),
	}
	m.mu.Lock()
	for i, sector := range m.sectors {
		db.Sectors[i] = &managerdb.Sector{
			Contract:   hex2bytes(sector.Contract),
			MerkleRoot: hex2bytes(sector.MerkleRoot),
			Data:       sector.Data,
		}
	}
	for _, set := range m.sets {
		set1 := &managerdb.Set{}
		for _, sector := range set.DataSectors {
			set1.DataIds = append(set1.DataIds, sector.id)
		}
		for _, sector := range set.ParitySectors {
			set1.ParityIds = append(set1.ParityIds, sector.id)
		}
		db.Sets = append(db.Sets, set1)
	}
	m.mu.Unlock()
	dump, err := proto.Marshal(db)
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
	f := func() (string, string, []byte, error) {
		m.mu.Lock()
		defer m.mu.Unlock()
		sector, has := m.sectors[i]
		if !has {
			return "", "", nil, fmt.Errorf("No such sector: %d", i)
		}
		if !sector.isPrimary {
			return "", "", nil, fmt.Errorf("The sector %d is not primary", i)
		}
		return sector.Contract, sector.MerkleRoot, sector.Data, nil
	}
	contract, sectorRoot, data, err := f()
	if err != nil {
		return nil, err
	}
	if data != nil {
		return data, nil
	}
	return m.siaclient.Read(contract, sectorRoot)
}

func (m *Manager) ReadSectorAt(i int64, offset, length int) ([]byte, error) {
	if offset < 0 || length < 0 || offset+length > m.sectorSize {
		return nil, fmt.Errorf("Bad range: [%d,%d)", offset, offset+length)
	}
	// TODO: read only needed part.
	data, err := m.ReadSector(i)
	if err != nil {
		return nil, err
	}
	return data[offset : offset+length], nil
}

func (m *Manager) AddSector(data []byte) (int64, error) {
	fmt.Printf("Manager.AddSector) start\n")
	defer fmt.Printf("Manager.AddSector) stop\n")
	if len(data) != m.sectorSize {
		return 0, fmt.Errorf("data length is %d", len(data))
	}
	m.mu.Lock()
	i := m.next
	m.next++
	sector := &Sector{
		Data:      data,
		id:        i,
		isPrimary: true,
	}
	m.sectors[i] = sector
	m.mu.Unlock()
	m.dataChan <- sector
	return i, nil
}

func (m *Manager) Start() error {
	go m.uploadPending()
	go func() {
		set := &Set{}
		for {
			select {
			case <-m.stopChan:
				break
			case sector := <-m.dataChan:
				set.DataSectors = append(set.DataSectors, sector)
				sector.set = set
				if len(set.DataSectors) == m.ndata {
					m.setChan <- set
					m.sets = append(m.sets, set)
					set = &Set{}
					log.Printf("Formed parity set.")
				}
			case set := <-m.setChan:
				m.handleSet(set)
			}
		}
		// Ingest all data from channels to unblock goroutines.
		for _ = range m.dataChan {
		}
		for _ = range m.setChan {
		}
	}()
	return nil
}

func (m *Manager) Stop() error {
	close(m.stopChan)
	return nil
}

func (m *Manager) uploadPending() {
	var sectors []*Sector
	sets := make(map[*Set]struct{})
	m.mu.Lock()
	for _, sector := range m.sectors {
		if sector.set == nil {
			sectors = append(sectors, sector)
		} else if len(sector.Contract) == 0 {
			sets[sector.set] = struct{}{}
		}
	}
	for _, set := range m.sets {
		if len(set.ParitySectors) == 0 {
			sets[set] = struct{}{}
		}
	}
	m.mu.Unlock()
	for _, sector := range sectors {
		log.Printf("Adding sector %d to dataChan.", sector.id)
		m.dataChan <- sector
	}
	for set := range sets {
		log.Printf("Handling a parity set.")
		m.setChan <- set
	}
}

func (m *Manager) handleSet(set *Set) {
	if len(set.ParitySectors) == 0 {
		if err := m.addParity(set); err != nil {
			log.Printf("m.addParity: %v.", err)
			go func() {
				time.Sleep(time.Second)
				m.setChan <- set
			}()
			return
		}
		log.Printf("Added parity sectors.")
	}
	if err := m.uploadSet(set); err != nil {
		log.Printf("m.uploadSet: %v.", err)
		go func() {
			time.Sleep(time.Second)
			m.setChan <- set
		}()
		return
	}
	log.Printf("Uploaded parity set.")
}

func (m *Manager) uploadSet(set *Set) error {
	all := append(set.DataSectors, set.ParitySectors...)
	used := make(map[string]struct{})
	for _, sector := range all {
		if len(sector.Contract) != 0 {
			used[sector.Contract] = struct{}{}
		}
	}
	n := len(all) - len(used)
	contracts, err := m.siaclient.Contracts()
	if err != nil {
		return fmt.Errorf("siaclient.Contracts: %v.", err)
	}
	var contracts1 []string
	for _, contract := range contracts {
		if _, has := used[contract]; !has {
			contracts1 = append(contracts1, contract)
		}
	}
	if len(contracts1) < n {
		return fmt.Errorf("too few contracts")
	}
	var chosen []string
	for _, i := range rand.Perm(len(contracts1))[:n] {
		chosen = append(chosen, contracts1[i])
	}
	for _, sector := range all {
		if len(sector.Contract) != 0 {
			continue
		}
		contract := chosen[0]
		chosen = chosen[1:]
		if err := m.uploadSector(sector, contract); err != nil {
			return fmt.Errorf("m.uploadSector: %v", err)
		}
	}
	return nil
}

func (m *Manager) addParity(set *Set) error {
	var datas [][]byte
	if len(set.DataSectors) != m.ndata {
		panic("len(set.DataSectors) != m.ndata")
	}
	for _, sector := range set.DataSectors {
		datas = append(datas, sector.Data)
	}
	for j := 0; j < m.nparity; j++ {
		datas = append(datas, make([]byte, m.sectorSize))
	}
	rs, err := reedsolomon.New(m.ndata, m.nparity)
	if err != nil {
		return fmt.Errorf("reedsolomon.New: %v", err)
	}
	if err := rs.Encode(datas); err != nil {
		return fmt.Errorf("rs.Encode: %v", err)
	}
	m.mu.Lock()
	for j := 0; j < m.nparity; j++ {
		i := m.next
		m.next++
		sector := &Sector{
			Data: datas[m.ndata+j],
			id:   i,
			set:  set,
		}
		m.sectors[i] = sector
		set.ParitySectors = append(set.ParitySectors, sector)
	}
	m.mu.Unlock()
	return nil
}

func (m *Manager) uploadSector(sector *Sector, contract string) error {
	sectorRoot, err := m.siaclient.Write(contract, sector.Data)
	if err != nil {
		return fmt.Errorf("siaclient.Write: %v.", err)
	}
	m.mu.Lock()
	sector.Contract = contract
	sector.MerkleRoot = sectorRoot
	sector.Data = nil
	m.mu.Unlock()
	return nil
}
