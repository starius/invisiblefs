package main

import (
	"flag"
	"log"

	"github.com/starius/invisiblefs/zipkv"
	"github.com/starius/invisiblefs/zipkv/fskv"
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
		log.Fatal("Provide -dir and -mountpoint")
	}
	dfs, err := fskv.New(*dir)
	if err != nil {
		log.Fatalf("Failed to create fskv object: %s.", err)
	}
	fe, err := zipkv.Zip(dfs, *bs)
	if err != nil {
		log.Fatalf("Failed to create zipkv object: %s.", err)
	}

	_ = fe
	// TODO: run HTTP server becked by fe.
}
