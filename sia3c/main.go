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
	dataDir    = flag.String("data-dir", "data-dir", "Directory to store databases")
	keyFile    = flag.String("key-file", "", "File with key ('disable' to disable encryption)")

	sc *siaclient.SiaClient
	ci manager.Cipher
	mn *manager.Manager
	fi *files.Files
	ks *kvsia.KvSia
)

type NoopCipher struct {
}

func (n *NoopCipher) Encrypt(sectorID int64, data []byte) {
}

func (n *NoopCipher) Decrypt(sectorID int64, data []byte) {
}

func main() {
	flag.Parse()
	if *keyFile == "" {
		log.Fatalf("Specify -key-file")
	} else if *keyFile == "disable" {
		ci = &NoopCipher{}
	} else {
		key, err := ioutil.ReadFile(*keyFile)
		if err != nil {
			log.Fatalf("ioutil.ReadFile(%q): %v.", *keyFile, err)
		}
		ci, err = crypto.New(key)
		if err != nil {
			log.Fatalf("crypto.New: %v.", err)
		}
	}
	mnFile := filepath.Join(*dataDir, "manager.db")
	fiFile := filepath.Join(*dataDir, "files.db")
	var err error
	sc, err = siaclient.New(*siaAddr, &http.Client{})
	if err != nil {
		log.Fatalf("siaclient.New: %v.", err)
	}
	if _, err := os.Stat(mnFile); !os.IsNotExist(err) {
		data, err := ioutil.ReadFile(mnFile)
		if err != nil {
			log.Fatalf("ioutil.ReadFile(%q): %v.", mnFile, err)
		}
		mn, err = manager.Load(data, sc, ci)
		if err != nil {
			log.Fatalf("manager.Load: %v.", err)
		}
	} else {
		mn, err = manager.New(*ndata, *nparity, *sectorSize, sc, ci)
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
	go func() {
	begin:
		time.Sleep(10 * time.Second)
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
		goto begin
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
	// Handle signals - close file.
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()
		for signal := range c {
			fmt.Printf("Caught %s.\n", signal)
			fmt.Printf("Writting the remaining files.\n")
			if err := ks.Sync(); err != nil {
				fmt.Printf("Failed to write: %s.\n", err)
				continue
			}
			fmt.Printf("Successfully written files.\n")
			// TODO: write manager and files dumps.
			fmt.Printf("Closing the listener on %s.\n", *addr)
			if err := ln.Close(); err != nil {
				fmt.Printf("Failed to close the listener: %s.\n", err)
				continue
			}
			fmt.Printf("Successfully closed the listener.\n")
			fmt.Printf("Exiting.\n")
			return
		}
	}()
	http.Handle("/", handler)
	log.Fatal(http.Serve(ln, nil))
}
