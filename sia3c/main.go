package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/starius/invisiblefs/kvsia"
	"github.com/starius/invisiblefs/siaform/cache"
	"github.com/starius/invisiblefs/siaform/crypto"
	"github.com/starius/invisiblefs/siaform/files"
	"github.com/starius/invisiblefs/siaform/manager"
	"github.com/starius/invisiblefs/siaform/siaclient"
	"github.com/starius/invisiblefs/zipkvserver/kvhttp"
)

var (
	addr   = flag.String("addr", "127.0.0.1:7712", "S3C HTTP server")
	bucket = flag.String("bucket", "bucket", "Bucket name")

	siaAddr    = flag.String("sia-addr", "127.0.0.1:9980", "Sia API addrer.")
	ndata      = flag.Int("ndata", 10, "Number of data sectors in a group")
	nparity    = flag.Int("nparity", 10, "Number of parity sectors in a group")
	sectorSize = flag.Int("sector-size", 4*1024*1024, "Sia block size")
	cacheSize  = flag.Int("cache-size", 100, "Size of LRU cache, in sectors")
	dataDir    = flag.String("data-dir", "data-dir", "Directory to store databases")
	keyFile    = flag.String("key-file", "", "File with key ('disable' to disable encryption)")

	mn *manager.Manager
	fi *files.Files
	ks *kvsia.KvSia
)

func main() {
	flag.Parse()
	mnFile := filepath.Join(*dataDir, "manager.db")
	fiFile := filepath.Join(*dataDir, "files.db")
	var err error
	var sc manager.SiaClient
	sc, err = siaclient.New(*siaAddr, &http.Client{})
	if err != nil {
		log.Fatalf("siaclient.New: %v.", err)
	}
	if *keyFile == "" {
		log.Fatalf("Specify -key-file")
	} else if *keyFile != "disable" {
		key, err := ioutil.ReadFile(*keyFile)
		if err != nil {
			log.Fatalf("ioutil.ReadFile(%q): %v.", *keyFile, err)
		}
		sc, err = crypto.New(key, sc)
		if err != nil {
			log.Fatalf("crypto.New: %v.", err)
		}
	}
	if *cacheSize > 0 {
		sc, err = cache.New(*cacheSize, sc)
		if err != nil {
			log.Fatalf("cache.New: %v.", err)
		}
	}
	if _, err := os.Stat(mnFile); !os.IsNotExist(err) {
		data, err := ioutil.ReadFile(mnFile)
		if err != nil {
			log.Fatalf("ioutil.ReadFile(%q): %v.", mnFile, err)
		}
		mn, err = manager.Load(data, sc)
		if err != nil {
			log.Fatalf("manager.Load: %v.", err)
		}
	} else {
		mn, err = manager.New(*ndata, *nparity, *sectorSize, sc)
		if err != nil {
			log.Fatalf("manager.New: %v.", err)
		}
	}
	if _, err := os.Stat(fiFile); !os.IsNotExist(err) {
		data, err := ioutil.ReadFile(fiFile)
		if err != nil {
			log.Fatalf("ioutil.ReadFile(%q): %v.", fiFile, err)
		}
		fi, err = files.Load(data, mn)
		if err != nil {
			log.Fatalf("files.Load: %v.", err)
		}
	} else {
		fi, err = files.New(*sectorSize, mn)
		if err != nil {
			log.Fatalf("files.New: %v.", err)
		}
	}
	if err := mn.Start(); err != nil {
		log.Fatalf("manager.Start: %v.", err)
	}
	var saveMu sync.Mutex
	save := func() {
		saveMu.Lock()
		defer saveMu.Unlock()
		data, err := mn.DumpDb()
		if err != nil {
			log.Fatalf("mn.DumpDb: %v.", err)
		}
		if err := ioutil.WriteFile(mnFile, data, 0600); err != nil {
			log.Fatalf("ioutil.WriteFile(%q, ...): %v.", mnFile, err)
		}
		data, err = fi.DumpDb()
		if err != nil {
			log.Fatalf("fi.DumpDb: %v.", err)
		}
		if err := ioutil.WriteFile(fiFile, data, 0600); err != nil {
			log.Fatalf("ioutil.WriteFile(%q, ...): %v.", fiFile, err)
		}
	}
	go func() {
		for {
			time.Sleep(10 * time.Second)
			save()
		}
	}()
	if ks, err = kvsia.New(fi); err != nil {
		log.Fatalf("kvsia.New: %v.", err)
	}
	handler, err := kvhttp.New(ks, *sectorSize, "/"+*bucket+"/")
	if err != nil {
		log.Fatalf("Failed to create kvhttp handler object: %s.", err)
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("Failed to listen: %s.", err)
	}
	finished := false
	var finishedMu sync.Mutex
	// Handle signals - close file.
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	var wg sync.WaitGroup
	defer wg.Wait()
	go func() {
		wg.Add(1)
		defer wg.Done()
		for signal := range c {
			fmt.Printf("Caught %s.\n", signal)
			//
			finishedMu.Lock()
			finished = true
			finishedMu.Unlock()
			//
			fmt.Printf("Closing the listener on %s.\n", *addr)
			if err := ln.Close(); err != nil {
				fmt.Printf("Failed to close the listener: %s.\n", err)
				continue
			}
			fmt.Printf("Successfully closed the listener.\n")
			//
			fmt.Printf("Saving local databases.\n")
			save()
			fmt.Printf("Successfully saved local databases.\n")
			//
			fmt.Printf("Sending sector in progress to manager.\n")
			if err := fi.UploadSectorInProgress(); err != nil {
				fmt.Printf("Failed to send sector in progress to manager: %s.\n", err)
				continue
			}
			fmt.Printf("Successfully sent sector in progress to manager.\n")
			//
			fmt.Printf("Sending pending sectors to upload.\n")
			mn.UploadAllPending()
			fmt.Printf("Successfully sent pending sectors to upload.\n")
			//
			fmt.Printf("Waiting for everything to upload.\n")
			mn.WaitForUploading()
			fmt.Printf("Successfully uploaded everything.\n")
			//
			fmt.Printf("Stopping manager.\n")
			if err := mn.Stop(); err != nil {
				fmt.Printf("Failed to stop the manager: %s.\n", err)
				continue
			}
			fmt.Printf("Successfully stopped manager.\n")
			//
			fmt.Printf("Saving local databases again.\n")
			save()
			fmt.Printf("Successfully saved local databases.\n")
			//
			fmt.Printf("Exiting.\n")
			return
		}
	}()
	http.Handle("/", handler)
	err = http.Serve(ln, nil)
	time.Sleep(time.Second)
	finishedMu.Lock()
	finished1 := finished
	finishedMu.Unlock()
	if finished1 {
		return
	}
	log.Fatal(err)
}
