package manager

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strings"
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

	isData bool
	set    *Set
	id     int64
}

type Set struct {
	DataSectors, ParitySectors []*Sector
}

type Cipher interface {
	Encrypt(sectorID int64, data []byte)
	Decrypt(sectorID int64, data []byte)
}

type Manager struct {
	sectors map[int64]*Sector
	sets    []*Set
	next    int64
	pending []*Sector
	mu      sync.Mutex

	siaclient *siaclient.SiaClient
	cipher    Cipher

	readsHistory   map[string]*managerdb.Latency
	readsHistoryMu sync.Mutex

	lastFailure   map[string]time.Time
	lastFailureMu sync.Mutex

	ndata, nparity, sectorSize int

	dataChan chan struct{}
	setChan  chan *Set
	stopChan chan struct{}
}

func New(ndata, nparity, sectorSize int, sc *siaclient.SiaClient, cipher Cipher) (*Manager, error) {
	return &Manager{
		sectors:      make(map[int64]*Sector),
		next:         1,
		ndata:        ndata,
		nparity:      nparity,
		sectorSize:   sectorSize,
		siaclient:    sc,
		cipher:       cipher,
		readsHistory: make(map[string]*managerdb.Latency),
		lastFailure:  make(map[string]time.Time),
		dataChan:     make(chan struct{}, 100),
		setChan:      make(chan *Set, 100),
		stopChan:     make(chan struct{}),
	}, nil
}

func Load(zdump []byte, sc *siaclient.SiaClient, cipher Cipher) (*Manager, error) {
	dump, err := gzip.Gunzip(zdump)
	if err != nil {
		return nil, fmt.Errorf("gzip.Gunzip(zdump): %v", err)
	}
	db := &managerdb.Db{}
	if err := proto.Unmarshal(dump, db); err != nil {
		return nil, fmt.Errorf("proto.Unmarshal(dump, db): %v", err)
	}
	m := &Manager{
		sectors:      make(map[int64]*Sector),
		siaclient:    sc,
		cipher:       cipher,
		readsHistory: db.ReadsHistory,
		lastFailure:  make(map[string]time.Time),
		dataChan:     make(chan struct{}, 100),
		setChan:      make(chan *Set, 100),
		stopChan:     make(chan struct{}),
		ndata:        int(db.Ndata),
		nparity:      int(db.Nparity),
		sectorSize:   int(db.SectorSize),
	}
	if m.readsHistory == nil {
		m.readsHistory = make(map[string]*managerdb.Latency)
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
			sector.isData = true
		}
		for _, i := range set.ParityIds {
			sector := m.sectors[i]
			set1.ParitySectors = append(set1.ParitySectors, sector)
			sector.set = set1
		}
		m.sets = append(m.sets, set1)
	}
	for _, i := range db.Pending {
		m.pending = append(m.pending, m.sectors[i])
		m.sectors[i].isData = true
	}
	return m, nil
}

func (m *Manager) DumpDb() ([]byte, error) {
	db := &managerdb.Db{
		Sectors:      make(map[int64]*managerdb.Sector),
		Ndata:        int32(m.ndata),
		Nparity:      int32(m.nparity),
		SectorSize:   int32(m.sectorSize),
		ReadsHistory: m.readsHistory,
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
	for _, sector := range m.pending {
		db.Pending = append(db.Pending, sector.id)
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

func (m *Manager) getSector(i int64) (string, string, []byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sector, has := m.sectors[i]
	if !has {
		return "", "", nil, fmt.Errorf("No such sector: %d", i)
	}
	if !sector.isData {
		return "", "", nil, fmt.Errorf("The sector %d is not data", i)
	}
	return sector.Contract, sector.MerkleRoot, sector.Data, nil
}

func (m *Manager) load(i int64, contract, sectorRoot string) ([]byte, error) {
	log.Printf("Loading data from contract %s", contract)
	if contract == "84e570089934203463967a7bf8b55b37664f597dc1004cb20f526d87259fecda" {
		return nil, fmt.Errorf("test error")
	}
	t1 := time.Now()
	data, err := m.siaclient.Read(contract, sectorRoot)
	latency := time.Since(t1)
	m.readsHistoryMu.Lock()
	l, has := m.readsHistory[contract]
	if !has {
		l = &managerdb.Latency{}
		m.readsHistory[contract] = l
	}
	l.TotalMs += latency.Nanoseconds() / 1e6
	l.Count++
	m.readsHistoryMu.Unlock()
	if len(data) != m.sectorSize {
		return nil, fmt.Errorf("Bad data length: %d. Want %d", len(data), m.sectorSize)
	}
	if err == nil {
		m.cipher.Decrypt(i, data)
	}
	return data, err
}

type event struct {
	j    int
	data []byte
}

func (m *Manager) recoverData(i int64) ([]byte, error) {
	m.mu.Lock()
	sector, has := m.sectors[i]
	if !has {
		m.mu.Unlock()
		return nil, fmt.Errorf("unknown sector %d", i)
	}
	set := sector.set
	if set == nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("sector %d not in set", i)
	}
	group := []Sector{}
	all := append(set.DataSectors, set.ParitySectors...)
	for _, sector := range all {
		group = append(group, Sector{
			id:         sector.id,
			Contract:   sector.Contract,
			MerkleRoot: sector.MerkleRoot,
			Data:       sector.Data,
		})
	}
	ndata := len(set.DataSectors)
	nparity := len(set.ParitySectors)
	m.mu.Unlock()
	events := make(chan event, ndata+nparity)
	for j, s := range group {
		if s.Data != nil {
			events <- event{j, s.Data}
			continue
		}
		go func(j int, s Sector) {
			data, err := m.load(s.id, s.Contract, s.MerkleRoot)
			if err == nil {
				events <- event{j, data}
			} else {
				log.Printf("Failed to load sector %d from %q", s.id, s.Contract)
				events <- event{j, nil}
			}
		}(j, s)
	}
	known := 0
	datas := make([][]byte, ndata+nparity)
	for k := 0; k < ndata+nparity; k++ {
		var e event
		select {
		case <-m.stopChan:
			return nil, fmt.Errorf("The manager was stopped")
		case e = <-events:
		}
		if e.data == nil {
			continue
		}
		datas[e.j] = e.data
		known++
		if known == ndata {
			break
		}
	}
	if known < ndata {
		return nil, fmt.Errorf("not enough data to recover sector %d", i)
	}
	rs, err := reedsolomon.New(ndata, nparity)
	if err != nil {
		return nil, fmt.Errorf("reedsolomon.New: %v", err)
	}
	if err := rs.Reconstruct(datas); err != nil {
		return nil, fmt.Errorf("rs.Reconstruct: %v", err)
	}
	thisJ := -1
	for j, s := range group {
		if s.id == i {
			thisJ = j
		}
	}
	if thisJ == -1 {
		panic("discrepancy: the sector must be in the set")
	}
	log.Printf("Recovered sector %d", i)
	return datas[thisJ], nil
}

func (m *Manager) ReadSector(i int64) ([]byte, error) {
	log.Printf("Reading sector %d", i)
	contract, sectorRoot, data, err := m.getSector(i)
	if err != nil {
		return nil, err
	}
	if data != nil {
		log.Printf("Sector %d is found in memory", i)
		return data, nil
	}
	data, err = m.load(i, contract, sectorRoot)
	if err == nil {
		return data, nil
	}
	log.Printf("Sector %d is broken - recovering", i)
	return m.recoverData(i)
}

func (m *Manager) InsecureReadSectorAt(i int64, offset, length int) ([]byte, error) {
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
	i, err := m.AllocateSector()
	if err != nil {
		return 0, fmt.Errorf("m.AllocateSector: %v", err)
	}
	if err := m.WriteSector(i, data); err != nil {
		return 0, fmt.Errorf("m.WriteSector(%d): %v", i, err)
	}
	return i, nil
}

func (m *Manager) AllocateSector() (int64, error) {
	m.mu.Lock()
	i := m.next
	m.next++
	m.sectors[i] = &Sector{}
	m.mu.Unlock()
	log.Printf("Allocated sector %d", i)
	return i, nil
}

func (m *Manager) WriteSector(i int64, data []byte) error {
	if len(data) != m.sectorSize {
		return fmt.Errorf("data length is %d", len(data))
	}
	m.mu.Lock()
	sector, has := m.sectors[i]
	if !has {
		m.mu.Unlock()
		return fmt.Errorf("sector %d not found", i)
	}
	if sector.Data != nil || sector.Contract != "" {
		m.mu.Unlock()
		return fmt.Errorf("sector %d is not empty", i)
	}
	sector.Data = data
	sector.id = i
	sector.isData = true
	m.pending = append(m.pending, sector)
	m.mu.Unlock()
	log.Printf("Filled sector with data: %d", i)
	m.dataChan <- struct{}{}
	return nil
}

func (m *Manager) Start() error {
	go m.continueUploads()
	go func() {
		for {
			select {
			case <-m.stopChan:
				break
			case <-m.dataChan:
				m.handlePending()
			case set := <-m.setChan:
				go m.handleSet(set)
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

func (m *Manager) continueUploads() {
	sets := make(map[*Set]struct{})
	m.mu.Lock()
	for _, sector := range m.sectors {
		if sector.set != nil && len(sector.Contract) == 0 {
			sets[sector.set] = struct{}{}
		}
		if sector.Data != nil {
			s := *sector
			s.Data = nil
			log.Printf("Sector with data: %#v", s)
		}
	}
	m.mu.Unlock()
	for set := range sets {
		log.Printf("Handling a parity set.")
		m.setChan <- set
	}
}

func (m *Manager) handlePending() {
	var newSets []*Set
	m.mu.Lock()
	for len(m.pending) >= m.ndata {
		set := &Set{
			DataSectors: m.pending[:m.ndata],
		}
		m.pending = m.pending[m.ndata:]
		m.addParity(set)
		for _, sector := range set.DataSectors {
			sector.set = set
		}
		for _, sector := range set.ParitySectors {
			sector.set = set
		}
		m.sets = append(m.sets, set)
		newSets = append(newSets, set)
		log.Printf("Formed parity set.")
	}
	m.mu.Unlock()
	for _, set := range newSets {
		m.setChan <- set
	}
}

func (m *Manager) handleSet(set *Set) {
	log.Printf("Uploading a parity set")
	if err := m.uploadSet(set); err != nil {
		log.Printf("m.uploadSet: %v.", err)
		time.Sleep(time.Second)
		m.setChan <- set
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
	var dataSectors, paritySectors []*Sector
	for _, sector := range all {
		if len(sector.Contract) != 0 {
			continue
		}
		if sector.isData {
			dataSectors = append(dataSectors, sector)
		} else {
			paritySectors = append(paritySectors, sector)
		}
	}
	contracts, err := m.siaclient.Contracts()
	if err != nil {
		return fmt.Errorf("siaclient.Contracts: %v.", err)
	}
	var contracts1 []string
	m.lastFailureMu.Lock()
	for _, contract := range contracts {
		if _, has := used[contract]; has {
			continue
		}
		if a, has := m.lastFailure[contract]; has && time.Since(a) < time.Minute {
			continue
		}
		contracts1 = append(contracts1, contract)
	}
	m.lastFailureMu.Unlock()
	if len(contracts1) < n {
		return fmt.Errorf("too few contracts")
	}
	m.readsHistoryMu.Lock()
	sort.Slice(contracts1, func(i, j int) bool {
		var iavg, javg int64
		il, has := m.readsHistory[contracts1[i]]
		if has {
			iavg = il.TotalMs / il.Count
		}
		jl, has := m.readsHistory[contracts1[j]]
		if has {
			javg = jl.TotalMs / jl.Count
		}
		return iavg < javg
	})
	m.readsHistoryMu.Unlock()
	var wg sync.WaitGroup
	errors := make(chan error, len(dataSectors)+len(paritySectors))
	// Upload data sectors to fastest contracts.
	for j, i := range rand.Perm(len(dataSectors)) {
		contract := contracts1[i]
		sector := dataSectors[j]
		wg.Add(1)
		go func(sector *Sector, contract string) {
			defer wg.Done()
			if err := m.uploadSector(sector, contract); err != nil {
				errors <- err
			}
		}(sector, contract)
	}
	// Upload parity sectors to random subset of other contracts.
	contracts1 = contracts1[len(dataSectors):]
	for j, i := range rand.Perm(len(contracts1))[:len(paritySectors)] {
		contract := contracts1[i]
		sector := paritySectors[j]
		wg.Add(1)
		go func(sector *Sector, contract string) {
			defer wg.Done()
			if err := m.uploadSector(sector, contract); err != nil {
				errors <- err
			}
		}(sector, contract)
	}
	wg.Wait()
	close(errors)
	var texts []string
	for err := range errors {
		texts = append(texts, err.Error())
	}
	if len(texts) > 0 {
		text := strings.Join(texts, "; ")
		return fmt.Errorf("m.uploadSector: %s", text)
	}
	return nil
}

func (m *Manager) addParity(set *Set) {
	// Run under m.mu.Lock().
	var datas [][]byte
	if len(set.DataSectors) != m.ndata {
		panic("len(set.DataSectors) != m.ndata")
	}
	for _, sector := range set.DataSectors {
		if len(sector.Data) != m.sectorSize {
			panic("len(sector.Data) != m.sectorSize")
		}
		datas = append(datas, sector.Data)
	}
	for j := 0; j < m.nparity; j++ {
		datas = append(datas, make([]byte, m.sectorSize))
	}
	rs, err := reedsolomon.New(m.ndata, m.nparity)
	if err != nil {
		panic(fmt.Sprintf("reedsolomon.New: %v", err))
	}
	if err := rs.Encode(datas); err != nil {
		panic(fmt.Sprintf("rs.Encode: %v", err))
	}
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
}

func (m *Manager) uploadSector(sector *Sector, contract string) error {
	data := make([]byte, m.sectorSize)
	copy(data, sector.Data)
	m.cipher.Encrypt(sector.id, data)
	sectorRoot, err := m.siaclient.Write(contract, data)
	if err != nil {
		m.lastFailureMu.Lock()
		m.lastFailure[contract] = time.Now()
		m.lastFailureMu.Unlock()
	}
	if err != nil {
		return fmt.Errorf("siaclient.Write(%q): %v.", contract, err)
	}
	m.mu.Lock()
	sector.Contract = contract
	sector.MerkleRoot = sectorRoot
	sector.Data = nil
	m.mu.Unlock()
	return nil
}
