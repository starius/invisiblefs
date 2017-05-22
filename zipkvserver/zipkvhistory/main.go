package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/starius/invisiblefs/zipkvserver/fskv"
	"github.com/starius/invisiblefs/zipkvserver/zipkv"
)

var (
	dir = flag.String("dir", "", "Dir with underlying files")
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
	blockSize := 1
	fe, err := zipkv.Zip(dfs, blockSize, -1)
	if err != nil {
		log.Fatalf("Failed to create zipkv object: %s.", err)
	}
	fmt.Printf("rev\toperation\tfilename\n")
	for i, change := range fe.History() {
		operation := "delete"
		if change.Put {
			operation = "put"
		}
		fmt.Printf("%d\t%s\t%s\n", i, operation, change.Filename)
	}
}
