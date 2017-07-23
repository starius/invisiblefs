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

type SiaClient interface {
	Contracts() ([]string, error)
	Read(contractID, sectorRoot string, sectorID int64) ([]byte, error)
	Write(contractID string, data []byte, sectorID int64) (string, error)
}

type Manager struct {
	db         *managerdb.Db
	next       int64
	sector2set map[int64]int
	mu         sync.Mutex // db.Sectors, db.Sets, db.Pending.

	siaclient SiaClient

	readsHistoryMu sync.Mutex

	lastFailure   map[string]time.Time
	lastFailureMu sync.Mutex

	ndata, nparity, sectorSize int

	dataChan chan struct{}
	setChan  chan int
	stopChan chan struct{}
	finChan  chan struct{}

	uploadingSectors   map[int64]struct{}
	uploadingSectorsMu sync.Mutex

	uploadingSets   map[int]struct{}
	uploadingSetsMu sync.Mutex
}

func New(ndata, nparity, sectorSize int, sc SiaClient) (*Manager, error) {
	db := &managerdb.Db{
		Sectors:      make(map[int64]*managerdb.Sector),
		Ndata:        int32(ndata),
		Nparity:      int32(nparity),
		SectorSize:   int32(sectorSize),
		ReadsHistory: make(map[string]*managerdb.Latency),
	}
	return &Manager{
		db:               db,
		next:             1,
		ndata:            ndata,
		nparity:          nparity,
		sectorSize:       sectorSize,
		siaclient:        sc,
		sector2set:       make(map[int64]int),
		lastFailure:      make(map[string]time.Time),
		dataChan:         make(chan struct{}, 100),
		setChan:          make(chan int, 100),
		stopChan:         make(chan struct{}),
		finChan:          make(chan struct{}),
		uploadingSectors: make(map[int64]struct{}),
		uploadingSets:    make(map[int]struct{}),
	}, nil
}

func Load(zdump []byte, sc SiaClient) (*Manager, error) {
	dump, err := gzip.Gunzip(zdump)
	if err != nil {
		return nil, fmt.Errorf("gzip.Gunzip(zdump): %v", err)
	}
	db := &managerdb.Db{}
	if err := proto.Unmarshal(dump, db); err != nil {
		return nil, fmt.Errorf("proto.Unmarshal(dump, db): %v", err)
	}
	m := &Manager{
		db:               db,
		siaclient:        sc,
		lastFailure:      make(map[string]time.Time),
		sector2set:       make(map[int64]int),
		dataChan:         make(chan struct{}, 100),
		setChan:          make(chan int, 100),
		stopChan:         make(chan struct{}),
		finChan:          make(chan struct{}),
		ndata:            int(db.Ndata),
		nparity:          int(db.Nparity),
		sectorSize:       int(db.SectorSize),
		uploadingSectors: make(map[int64]struct{}),
		uploadingSets:    make(map[int]struct{}),
	}
	if m.db.Sectors == nil {
		m.db.Sectors = make(map[int64]*managerdb.Sector)
	}
	if m.db.ReadsHistory == nil {
		m.db.ReadsHistory = make(map[string]*managerdb.Latency)
	}
	maxI := int64(0)
	for i := range m.db.Sectors {
		if i > maxI {
			maxI = i
		}
	}
	m.next = maxI + 1
	for j, set := range m.db.Sets {
		for _, i := range set.DataIds {
			m.sector2set[i] = j
		}
		for _, i := range set.ParityIds {
			m.sector2set[i] = j
		}
	}
	return m, nil
}

func (m *Manager) DumpDb() ([]byte, error) {
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

func (m *Manager) getSector(i int64) (string, string, []byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sector, has := m.db.Sectors[i]
	if !has {
		return "", "", nil, fmt.Errorf("No such sector: %d", i)
	}
	return bytes2hex(sector.Contract), bytes2hex(sector.MerkleRoot), sector.Data, nil
}

func (m *Manager) load(i int64, contract, sectorRoot string) ([]byte, error) {
	log.Printf("Loading data from contract %s", contract)
	if contract == "84e570089934203463967a7bf8b55b37664f597dc1004cb20f526d87259fecda" {
		return nil, fmt.Errorf("test error")
	}
	t1 := time.Now()
	data, err := m.siaclient.Read(contract, sectorRoot, i)
	latency := time.Since(t1)
	m.readsHistoryMu.Lock()
	l, has := m.db.ReadsHistory[contract]
	if !has {
		l = &managerdb.Latency{}
		m.db.ReadsHistory[contract] = l
	}
	l.TotalMs += latency.Nanoseconds() / 1e6
	l.Count++
	m.readsHistoryMu.Unlock()
	if len(data) != m.sectorSize {
		return nil, fmt.Errorf("Bad data length: %d. Want %d", len(data), m.sectorSize)
	}
	return data, err
}

type sectorData struct {
	id         int64
	contract   string
	merkleRoot string
	data       []byte
}

type event struct {
	j    int
	data []byte
}

func (m *Manager) recoverData(i int64) ([]byte, error) {
	m.mu.Lock()
	_, has := m.db.Sectors[i]
	if !has {
		m.mu.Unlock()
		return nil, fmt.Errorf("unknown sector %d", i)
	}
	setIndex, has := m.sector2set[i]
	if !has {
		m.mu.Unlock()
		return nil, fmt.Errorf("sector %d not in set", i)
	}
	set := m.db.Sets[setIndex]
	group := []sectorData{}
	all := append(set.DataIds, set.ParityIds...)
	for _, si := range all {
		sector := m.db.Sectors[si]
		group = append(group, sectorData{
			id:         si,
			contract:   bytes2hex(sector.Contract),
			merkleRoot: bytes2hex(sector.MerkleRoot),
			data:       sector.Data,
		})
	}
	ndata := len(set.DataIds)
	nparity := len(set.ParityIds)
	m.mu.Unlock()
	events := make(chan event, ndata+nparity)
	for j, s := range group {
		if s.data != nil {
			events <- event{j, s.data}
			continue
		}
		go func(j int, s sectorData) {
			data, err := m.load(s.id, s.contract, s.merkleRoot)
			if err == nil {
				events <- event{j, data}
			} else {
				log.Printf("Failed to load sector %d from %q", s.id, s.contract)
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
	m.db.Sectors[i] = &managerdb.Sector{}
	m.mu.Unlock()
	log.Printf("Allocated sector %d", i)
	return i, nil
}

func (m *Manager) WriteSector(i int64, data []byte) error {
	if len(data) != m.sectorSize {
		return fmt.Errorf("data length is %d", len(data))
	}
	m.mu.Lock()
	sector, has := m.db.Sectors[i]
	if !has {
		m.mu.Unlock()
		return fmt.Errorf("sector %d not found", i)
	}
	if len(sector.Data) != 0 || len(sector.Contract) != 0 {
		m.mu.Unlock()
		return fmt.Errorf("sector %d is not empty", i)
	}
	sector.Data = data
	m.db.Pending = append(m.db.Pending, i)
	m.mu.Unlock()
	log.Printf("Filled sector with data: %d", i)
	m.dataChan <- struct{}{}
	return nil
}

func (m *Manager) Start() error {
	go m.continueUploads()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		stopped := false
		for !stopped {
			select {
			case <-m.stopChan:
				stopped = true
			case <-m.dataChan:
				m.handlePending()
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		stopped := false
		for !stopped {
			select {
			case <-m.stopChan:
				stopped = true
			case setIndex := <-m.setChan:
				go m.handleSet(setIndex)
			}
		}
	}()
	go func() {
		wg.Wait()
		// Ingest all data from channels to unblock goroutines.
		finished := false
		for !finished {
			select {
			case <-m.dataChan:
			case <-m.setChan:
			default:
				finished = true
			}
		}
		close(m.finChan)
	}()
	return nil
}

func (m *Manager) Stop() error {
	close(m.stopChan)
	<-m.finChan
	return nil
}

func (m *Manager) UploadAllPending() {
	var newSets []int
	m.mu.Lock()
	for len(m.db.Pending) > 0 {
		newSets = append(newSets, m.formParitySet())
	}
	m.mu.Unlock()
	if len(newSets) == 0 {
		return
	}
	for _, setIndex := range newSets {
		m.setChan <- setIndex
		log.Printf("Uploading incomplete parity set from pending")
	}
}

func (m *Manager) WaitForUploading() {
	log.Printf("Waiting for the incomplete parity sets to upload.")
	for {
		time.Sleep(time.Second)
		allUploaded := true
		m.mu.Lock()
		for _, set := range m.db.Sets {
			for _, si := range set.DataIds {
				sector := m.db.Sectors[si]
				if len(sector.Contract) == 0 {
					allUploaded = false
				}
			}
			for _, si := range set.ParityIds {
				sector := m.db.Sectors[si]
				if len(sector.Contract) == 0 {
					allUploaded = false
				}
			}
		}
		m.mu.Unlock()
		if allUploaded {
			log.Printf("All sectors were uploaded.")
			return
		} else {
			log.Printf("Still uploading pending sectors.")
		}
	}
}

func (m *Manager) continueUploads() {
	sets := make(map[int]struct{})
	m.mu.Lock()
	for si, sector := range m.db.Sectors {
		setIndex, has := m.sector2set[si]
		if has && len(sector.Contract) == 0 {
			sets[setIndex] = struct{}{}
		}
	}
	m.mu.Unlock()
	for setIndex := range sets {
		log.Printf("Handling a parity set.")
		m.setChan <- setIndex
	}
}

func (m *Manager) handlePending() {
	var newSets []int
	m.mu.Lock()
	for len(m.db.Pending) >= m.ndata {
		newSets = append(newSets, m.formParitySet())
	}
	m.mu.Unlock()
	for _, setIndex := range newSets {
		m.setChan <- setIndex
	}
}

func (m *Manager) formParitySet() int {
	// Run under m.mu.Lock().
	ndata := m.ndata
	if ndata > len(m.db.Pending) {
		ndata = len(m.db.Pending)
	}
	if ndata == 0 {
		panic("ndata == 0")
	}
	set := &managerdb.Set{
		DataIds: make([]int64, ndata),
	}
	// Don't put slice of Pending to set.DataIds because both of
	// them can modify it with append corrupting data of each other.
	copy(set.DataIds, m.db.Pending[:ndata])
	m.db.Pending = m.db.Pending[ndata:]
	// Add sectors to the set.
	setIndex := len(m.db.Sets)
	m.db.Sets = append(m.db.Sets, set)
	m.addParity(setIndex)
	for _, si := range set.DataIds {
		m.sector2set[si] = setIndex
	}
	for _, si := range set.ParityIds {
		m.sector2set[si] = setIndex
	}
	log.Printf("Formed parity set.")
	return setIndex
}

func (m *Manager) handleSet(setIndex int) {
	log.Printf("Uploading a parity set")
	if err := m.uploadSet(setIndex); err != nil {
		log.Printf("m.uploadSet: %v.", err)
		time.Sleep(time.Second)
		m.setChan <- setIndex
		return
	}
	log.Printf("Uploaded parity set.")
}

type indexedSector struct {
	sector *managerdb.Sector
	id     int64
}

func (m *Manager) uploadSet(setIndex int) error {
	m.uploadingSetsMu.Lock()
	if _, has := m.uploadingSets[setIndex]; has {
		panic(fmt.Sprintf("parallel uploadSet(%d)", setIndex))
	}
	m.uploadingSets[setIndex] = struct{}{}
	m.uploadingSetsMu.Unlock()
	defer func() {
		m.uploadingSetsMu.Lock()
		delete(m.uploadingSets, setIndex)
		m.uploadingSetsMu.Unlock()
	}()
	//
	var dataSectors0, paritySectors0 []indexedSector
	m.mu.Lock()
	set := m.db.Sets[setIndex]
	for _, si := range set.DataIds {
		dataSectors0 = append(dataSectors0, indexedSector{m.db.Sectors[si], si})
	}
	for _, si := range set.ParityIds {
		paritySectors0 = append(paritySectors0, indexedSector{m.db.Sectors[si], si})
	}
	m.mu.Unlock()
	used := make(map[string]struct{})
	for _, is := range dataSectors0 {
		if len(is.sector.Contract) != 0 {
			used[bytes2hex(is.sector.Contract)] = struct{}{}
		}
	}
	for _, is := range paritySectors0 {
		if len(is.sector.Contract) != 0 {
			used[bytes2hex(is.sector.Contract)] = struct{}{}
		}
	}
	n := len(dataSectors0) + len(paritySectors0) - len(used)
	var dataSectors, paritySectors []indexedSector
	for _, is := range dataSectors0 {
		if len(bytes2hex(is.sector.Contract)) != 0 {
			continue
		}
		if len(is.sector.Data) != m.sectorSize {
			return fmt.Errorf("uploadSet: len(sector.Data) is %d, want %d; sector %d", len(is.sector.Data), m.sectorSize, is.id)
		}
		dataSectors = append(dataSectors, is)
	}
	for _, is := range paritySectors0 {
		if len(bytes2hex(is.sector.Contract)) != 0 {
			continue
		}
		if len(is.sector.Data) != m.sectorSize {
			return fmt.Errorf("uploadSet: len(sector.Data) is %d, want %d; sector %d", len(is.sector.Data), m.sectorSize, is.id)
		}
		paritySectors = append(paritySectors, is)
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
		return fmt.Errorf("too few contracts (%d < %d)", len(contracts1), n)
	}
	m.readsHistoryMu.Lock()
	sort.Slice(contracts1, func(i, j int) bool {
		var iavg, javg int64
		il, has := m.db.ReadsHistory[contracts1[i]]
		if has {
			iavg = il.TotalMs / il.Count
		}
		jl, has := m.db.ReadsHistory[contracts1[j]]
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
		is := dataSectors[j]
		wg.Add(1)
		go func(is indexedSector, contract string) {
			defer wg.Done()
			if err := m.uploadSector(is.sector, is.id, contract); err != nil {
				errors <- fmt.Errorf("data sector: %v", err)
			}
		}(is, contract)
	}
	// Upload parity sectors to random subset of other contracts.
	contracts1 = contracts1[len(dataSectors):]
	for j, i := range rand.Perm(len(contracts1))[:len(paritySectors)] {
		contract := contracts1[i]
		is := paritySectors[j]
		wg.Add(1)
		go func(is indexedSector, contract string) {
			defer wg.Done()
			if err := m.uploadSector(is.sector, is.id, contract); err != nil {
				errors <- fmt.Errorf("parity sector: %v", err)
			}
		}(is, contract)
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

func (m *Manager) addParity(setIndex int) {
	// Run under m.mu.Lock().
	if m.nparity == 0 {
		return
	}
	set := m.db.Sets[setIndex]
	var datas [][]byte
	if len(set.DataIds) == 0 {
		panic("len(set.DataIds) == 0")
	}
	ndata := len(set.DataIds)
	for _, si := range set.DataIds {
		sector := m.db.Sectors[si]
		if len(sector.Data) != m.sectorSize {
			panic(fmt.Sprintf("sector %d: len(sector.Data) is %d, want %d", si, len(sector.Data), m.sectorSize))
		}
		datas = append(datas, sector.Data)
	}
	for j := 0; j < m.nparity; j++ {
		datas = append(datas, make([]byte, m.sectorSize))
	}
	rs, err := reedsolomon.New(ndata, m.nparity)
	if err != nil {
		panic(fmt.Sprintf("reedsolomon.New: %v", err))
	}
	if err := rs.Encode(datas); err != nil {
		panic(fmt.Sprintf("rs.Encode: %v", err))
	}
	for j := 0; j < m.nparity; j++ {
		i := m.next
		m.next++
		sector := &managerdb.Sector{
			Data: datas[ndata+j],
		}
		m.sector2set[i] = setIndex
		if len(sector.Data) != m.sectorSize {
			panic(fmt.Sprintf("len(sector.Data) is %d, want %d", len(sector.Data), m.sectorSize))
		}
		m.db.Sectors[i] = sector
		set.ParityIds = append(set.ParityIds, i)
	}
}

func (m *Manager) uploadSector(sector *managerdb.Sector, id int64, contract string) error {
	m.uploadingSectorsMu.Lock()
	if _, has := m.uploadingSectors[id]; has {
		panic(fmt.Sprintf("parallel uploadSector(%d)", id))
	}
	m.uploadingSectors[id] = struct{}{}
	m.uploadingSectorsMu.Unlock()
	defer func() {
		m.uploadingSectorsMu.Lock()
		delete(m.uploadingSectors, id)
		m.uploadingSectorsMu.Unlock()
	}()
	//
	if len(sector.Data) != m.sectorSize {
		return fmt.Errorf("uploadSector: len(sector.Data) is %d, want %d; sector %d", len(sector.Data), m.sectorSize, id)
	}
	sectorRoot, err := m.siaclient.Write(contract, sector.Data, id)
	if err != nil {
		m.lastFailureMu.Lock()
		m.lastFailure[contract] = time.Now()
		m.lastFailureMu.Unlock()
		return fmt.Errorf("siaclient.Write(%q): %v.", contract, err)
	}
	m.mu.Lock()
	sector.Contract = hex2bytes(contract)
	sector.MerkleRoot = hex2bytes(sectorRoot)
	sector.Data = nil
	m.mu.Unlock()
	return nil
}
