package manager

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

const (
	testSectorSize = 4096
)

type MockSiaClient struct {
	working    map[string]bool
	data       map[string][]byte
	sectorSize int
	mu         sync.Mutex
}

func NewMSC(sectorSize int) *MockSiaClient {
	return &MockSiaClient{
		working:    make(map[string]bool),
		data:       make(map[string][]byte),
		sectorSize: sectorSize,
	}
}

func (m *MockSiaClient) addContract(contractID string, working bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, has := m.working[contractID]; has {
		panic("addContract called twice on " + contractID)
	}
	m.working[contractID] = working
}

func (m *MockSiaClient) enable(contractID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, has := m.working[contractID]; !has {
		panic("enable called on unknown contract " + contractID)
	} else if state == true {
		panic("enable called on enabled contract " + contractID)
	}
	m.working[contractID] = true
}

func (m *MockSiaClient) disable(contractID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if state, has := m.working[contractID]; !has {
		panic("enable called on unknown contract " + contractID)
	} else if state == false {
		panic("enable called on disabled contract " + contractID)
	}
	m.working[contractID] = false
}

func (m *MockSiaClient) Contracts() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var contracts []string
	for contractID := range m.working {
		contracts = append(contracts, contractID)
	}
	return contracts, nil
}

func (m *MockSiaClient) Read(contractID, sectorRoot string, sectorID int64) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if working, has := m.working[contractID]; !has {
		return nil, fmt.Errorf("Unknown contractID: %q", contractID)
	} else if !working {
		return nil, fmt.Errorf("contract %q is down", contractID)
	}
	if data, has := m.data[contractID+"-"+sectorRoot]; !has {
		return nil, fmt.Errorf("sector %q doesn't exist", sectorRoot)
	} else {
		return data, nil
	}
}

func (m *MockSiaClient) Write(contractID string, data []byte, sectorID int64) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if working, has := m.working[contractID]; !has {
		return "", fmt.Errorf("Unknown contractID: %q", contractID)
	} else if !working {
		return "", fmt.Errorf("contract %q is down", contractID)
	}
	if len(data) != m.sectorSize {
		return "", fmt.Errorf("len(data) is %d, want %d", len(data), m.sectorSize)
	}
	checksum := sha256.Sum256(data)
	sectorRoot := hex.EncodeToString(checksum[:])
	m.data[contractID+"-"+sectorRoot] = data
	return sectorRoot, nil
}

func makeData(i, sectorSize int) []byte {
	source := rand.NewSource(int64(i))
	r := rand.New(source)
	data := make([]byte, sectorSize)
	if n, err := r.Read(data); err != nil {
		panic(err)
	} else if n != sectorSize {
		panic(n != sectorSize)
	}
	return data
}

func TestUploadAndDownloadOneSector(t *testing.T) {
	sc := NewMSC(testSectorSize)
	mn, err := New(1, 1, testSectorSize, sc)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := mn.Start(); err != nil {
		t.Fatalf("mn.Start: %v", err)
	}
	defer func() {
		if err := mn.Stop(); err != nil {
			t.Fatalf("mn.Stop: %v", err)
		}
	}()
	//
	data0 := makeData(1, testSectorSize)
	i, err := mn.AddSector(data0)
	if err != nil {
		t.Fatalf("mn.AddSector: %v", err)
	}
	data, err := mn.ReadSector(i)
	if err != nil {
		t.Fatalf("mn.ReadSector: %v", err)
	}
	if !bytes.Equal(data0, data) {
		t.Errorf("data != data0")
	}
}

func TestUploadAndDownloadManySectors(t *testing.T) {
	sc := NewMSC(testSectorSize)
	mn, err := New(1, 1, testSectorSize, sc)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := mn.Start(); err != nil {
		t.Fatalf("mn.Start: %v", err)
	}
	defer func() {
		if err := mn.Stop(); err != nil {
			t.Fatalf("mn.Stop: %v", err)
		}
	}()
	//
	var ids []int64
	for k := 0; k < 100; k++ {
		data0 := makeData(k, testSectorSize)
		i, err := mn.AddSector(data0)
		if err != nil {
			t.Fatalf("mn.AddSector: %v", err)
		}
		ids = append(ids, i)
	}
	for k := 0; k < 100; k++ {
		data0 := makeData(k, testSectorSize)
		data, err := mn.ReadSector(ids[k])
		if err != nil {
			t.Fatalf("mn.ReadSector: %v", err)
		}
		if !bytes.Equal(data0, data) {
			t.Errorf("data != data0")
		}
	}
}

func TestUploadAndDownloadOneSectorWithContract(t *testing.T) {
	sc := NewMSC(testSectorSize)
	sc.addContract("01", true)
	mn, err := New(1, 0, testSectorSize, sc)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := mn.Start(); err != nil {
		t.Fatalf("mn.Start: %v", err)
	}
	//
	data0 := makeData(1, testSectorSize)
	i, err := mn.AddSector(data0)
	if err != nil {
		t.Fatalf("mn.AddSector: %v", err)
	}
	// Make sure that the data was uploaded.
	mn.WaitForUploading()
	if err := mn.Stop(); err != nil {
		t.Fatalf("mn.Stop: %v", err)
	}
	dump, err := mn.DumpDb()
	if err != nil {
		t.Fatalf("mn.DumpDb: %v", err)
	}
	if len(dump) > 100 {
		t.Fatalf("Dump is too large. The data was not uploaded.")
	}
	mn1, err := Load(dump, sc)
	if err != nil {
		t.Fatalf("mn1.Load: %v", err)
	}
	if err := mn1.Start(); err != nil {
		t.Fatalf("mn1.Start: %v", err)
	}
	//
	data, err := mn1.ReadSector(i)
	if err != nil {
		t.Fatalf("mn1.ReadSector: %v", err)
	}
	if !bytes.Equal(data0, data) {
		t.Errorf("data != data0")
	}
	//
	if err := mn1.Stop(); err != nil {
		t.Fatalf("mn1.Stop: %v", err)
	}
}

func TestUploadAndDownloadManySectorsWithContract(t *testing.T) {
	sc := NewMSC(testSectorSize)
	sc.addContract("01", true)
	mn, err := New(1, 0, testSectorSize, sc)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := mn.Start(); err != nil {
		t.Fatalf("mn.Start: %v", err)
	}
	//
	var ids []int64
	for k := 0; k < 100; k++ {
		data0 := makeData(k, testSectorSize)
		i, err := mn.AddSector(data0)
		if err != nil {
			t.Fatalf("mn.AddSector: %v", err)
		}
		ids = append(ids, i)
	}
	// Make sure that the data was uploaded.
	mn.WaitForUploading()
	if err := mn.Stop(); err != nil {
		t.Fatalf("mn.Stop: %v", err)
	}
	dump, err := mn.DumpDb()
	if err != nil {
		t.Fatalf("mn.DumpDb: %v", err)
	}
	if len(dump) > 10000 {
		t.Fatalf("Dump is too large: %d", len(dump))
	}
	mn1, err := Load(dump, sc)
	if err != nil {
		t.Fatalf("mn1.Load: %v", err)
	}
	if err := mn1.Start(); err != nil {
		t.Fatalf("mn1.Start: %v", err)
	}
	//
	for k := 0; k < 100; k++ {
		data0 := makeData(k, testSectorSize)
		data, err := mn.ReadSector(ids[k])
		if err != nil {
			t.Fatalf("mn.ReadSector: %v", err)
		}
		if !bytes.Equal(data0, data) {
			t.Errorf("data != data0")
		}
	}
	//
	if err := mn1.Stop(); err != nil {
		t.Fatalf("mn1.Stop: %v", err)
	}
}

func TestAllocate(t *testing.T) {
	sc := NewMSC(testSectorSize)
	mn, err := New(1, 1, testSectorSize, sc)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := mn.Start(); err != nil {
		t.Fatalf("mn.Start: %v", err)
	}
	defer func() {
		if err := mn.Stop(); err != nil {
			t.Fatalf("mn.Stop: %v", err)
		}
	}()
	//
	i, err := mn.AllocateSector()
	if err != nil {
		t.Fatalf("mn.AllocateSector: %v", err)
	}
	data0 := makeData(1, testSectorSize)
	if err := mn.WriteSector(i, data0); err != nil {
		t.Fatalf("mn.WriteSector: %v", err)
	}
	data, err := mn.ReadSector(i)
	if err != nil {
		t.Fatalf("mn.ReadSector: %v", err)
	}
	if !bytes.Equal(data0, data) {
		t.Errorf("data != data0")
	}
}

func TestAllocateAcrossDump(t *testing.T) {
	sc := NewMSC(testSectorSize)
	mn, err := New(1, 0, testSectorSize, sc)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := mn.Start(); err != nil {
		t.Fatalf("mn.Start: %v", err)
	}
	//
	i, err := mn.AllocateSector()
	if err != nil {
		t.Fatalf("mn.AllocateSector: %v", err)
	}
	data0 := makeData(1, testSectorSize)
	if err := mn.WriteSector(i, data0); err != nil {
		t.Fatalf("mn.WriteSector: %v", err)
	}
	//
	if err := mn.Stop(); err != nil {
		t.Fatalf("mn.Stop: %v", err)
	}
	dump, err := mn.DumpDb()
	if err != nil {
		t.Fatalf("mn.DumpDb: %v", err)
	}
	mn1, err := Load(dump, sc)
	if err != nil {
		t.Fatalf("mn1.Load: %v", err)
	}
	if err := mn1.Start(); err != nil {
		t.Fatalf("mn1.Start: %v", err)
	}
	//
	data, err := mn1.ReadSector(i)
	if err != nil {
		t.Fatalf("mn1.ReadSector: %v", err)
	}
	if !bytes.Equal(data0, data) {
		t.Errorf("data != data0")
	}
	//
	if err := mn1.Stop(); err != nil {
		t.Fatalf("mn1.Stop: %v", err)
	}
}

func TestUploadAllPending(t *testing.T) {
	sc := NewMSC(testSectorSize)
	for c := 1; c <= 7; c++ {
		sc.addContract(fmt.Sprintf("0%d", c), true)
	}
	mn, err := New(3, 4, testSectorSize, sc)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := mn.Start(); err != nil {
		t.Fatalf("mn.Start: %v", err)
	}
	//
	var ids []int64
	for k := 0; k < 100; k++ {
		data0 := makeData(k, testSectorSize)
		i, err := mn.AddSector(data0)
		if err != nil {
			t.Fatalf("mn.AddSector: %v", err)
		}
		ids = append(ids, i)
	}
	// Make sure that the data was uploaded.
	mn.WaitForUploading()
	mn.UploadAllPending()
	mn.WaitForUploading()
	if err := mn.Stop(); err != nil {
		t.Fatalf("mn.Stop: %v", err)
	}
	dump, err := mn.DumpDb()
	if err != nil {
		t.Fatalf("mn.DumpDb: %v", err)
	}
	if len(dump) > 100000 {
		t.Fatalf("Dump is too large: %d", len(dump))
	}
	mn1, err := Load(dump, sc)
	if err != nil {
		t.Fatalf("mn1.Load: %v", err)
	}
	if err := mn1.Start(); err != nil {
		t.Fatalf("mn1.Start: %v", err)
	}
	//
	for k := 0; k < 100; k++ {
		data0 := makeData(k, testSectorSize)
		data, err := mn.ReadSector(ids[k])
		if err != nil {
			t.Fatalf("mn.ReadSector: %v", err)
		}
		if !bytes.Equal(data0, data) {
			t.Errorf("data != data0")
		}
	}
	//
	if err := mn1.Stop(); err != nil {
		t.Fatalf("mn1.Stop: %v", err)
	}
}

func TestAllocateAndUploadAllPending(t *testing.T) {
	sc := NewMSC(testSectorSize)
	for c := 1; c <= 7; c++ {
		sc.addContract(fmt.Sprintf("0%d", c), true)
	}
	mn, err := New(3, 4, testSectorSize, sc)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := mn.Start(); err != nil {
		t.Fatalf("mn.Start: %v", err)
	}
	//
	_, err = mn.AllocateSector()
	if err != nil {
		t.Fatalf("mn.AddSector: %v", err)
	}
	// Make sure WaitForUploading and UploadAllPending ignore
	// empty allocated blocks.
	mn.WaitForUploading()
	mn.UploadAllPending()
	mn.WaitForUploading()
	mn.UploadAllPending()
	//
	if err := mn.Stop(); err != nil {
		t.Fatalf("mn.Stop: %v", err)
	}
	dump, err := mn.DumpDb()
	if err != nil {
		t.Fatalf("mn.DumpDb: %v", err)
	}
	if len(dump) > 100 {
		t.Fatalf("Dump is too large: %d", len(dump))
	}
	mn1, err := Load(dump, sc)
	if err != nil {
		t.Fatalf("mn1.Load: %v", err)
	}
	if err := mn1.Start(); err != nil {
		t.Fatalf("mn1.Start: %v", err)
	}
	//
	// Make sure WaitForUploading and UploadAllPending ignore
	// empty allocated blocks. Also after dump-load cycle.
	mn.WaitForUploading()
	mn.UploadAllPending()
	mn.WaitForUploading()
	mn.UploadAllPending()
	//
	if err := mn1.Stop(); err != nil {
		t.Fatalf("mn1.Stop: %v", err)
	}
}

func TestRecover(t *testing.T) {
	sc := NewMSC(testSectorSize)
	for c := 1; c <= 7; c++ {
		sc.addContract(fmt.Sprintf("0%d", c), true)
	}
	mn, err := New(3, 4, testSectorSize, sc)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := mn.Start(); err != nil {
		t.Fatalf("mn.Start: %v", err)
	}
	//
	var ids []int64
	for k := 0; k < 100; k++ {
		data0 := makeData(k, testSectorSize)
		i, err := mn.AddSector(data0)
		if err != nil {
			t.Fatalf("mn.AddSector: %v", err)
		}
		ids = append(ids, i)
	}
	//
	mn.WaitForUploading()
	sc.disable("01")
	sc.disable("03")
	sc.disable("05")
	sc.disable("07")
	//
	for k := 0; k < 100; k++ {
		data0 := makeData(k, testSectorSize)
		data, err := mn.ReadSector(ids[k])
		if err != nil {
			t.Fatalf("mn.ReadSector: %v", err)
		}
		if !bytes.Equal(data0, data) {
			t.Errorf("data != data0")
		}
	}
	//
	if err := mn.Stop(); err != nil {
		t.Fatalf("mn.Stop: %v", err)
	}
}
