package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/starius/invisiblefs/zipkvserver/fskv"
	"github.com/starius/invisiblefs/zipkvserver/kvhttp"
	"github.com/starius/invisiblefs/zipkvserver/zipkv"
)

var (
	dir  = flag.String("dir", "", "Dir with underlying files")
	addr = flag.String("addr", "127.0.0.1:7711", "HTTP server")
	bs   = flag.Int("bs", 40*1024*1024, "Block size, bytes")
)

func main() {
	flag.Parse()
	if *dir == "" {
		flag.PrintDefaults()
		log.Fatal("Provide -dir.")
	}
	dfs, err := fskv.New(*dir)
	if err != nil {
		log.Fatalf("Failed to create fskv object: %s.", err)
	}
	fe, err := zipkv.Zip(dfs, *bs)
	if err != nil {
		log.Fatalf("Failed to create zipkv object: %s.", err)
	}
	handler, err := kvhttp.New(fe, *bs)
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
			fmt.Printf("Writting the remaining files to %s.\n", *dir)
			if err := fe.Sync(); err != nil {
				fmt.Printf("Failed to write: %s.\n", err)
				continue
			}
			fmt.Printf("Successfully written files.\n")
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
