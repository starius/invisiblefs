package main

import (
	"flag"
	"log"

	"github.com/starius/invisiblefs/zipkvserver/fskv"
	"github.com/starius/invisiblefs/zipkvserver/zipkv"
)

var (
	from = flag.String("from", "", "Source directory")
	to   = flag.String("to", "", "Destination directory")
	toBs = flag.Int("to-bs", 40*1024*1024, "Destination block size")
	rev  = flag.Int("rev", -1, "Stop after this rev (-1 = copy all)")
)

func main() {
	flag.Parse()
	if *from == "" || *to == "" {
		flag.PrintDefaults()
		log.Fatal("Provide -from and -to.")
	}
	fromFs, err := fskv.New(*from)
	if err != nil {
		log.Fatalf("Failed to create fskv object: %s.", err)
	}
	toFs, err := fskv.New(*to)
	if err != nil {
		log.Fatalf("Failed to create fskv object: %s.", err)
	}
	fromBlockSize := 1
	fromFe, err := zipkv.Zip(fromFs, fromBlockSize, *rev)
	if err != nil {
		log.Fatalf("Failed to create zipkv object: %s.", err)
	}
	toFe, err := zipkv.Zip(toFs, *toBs, -1)
	if err != nil {
		log.Fatalf("Failed to create zipkv object: %s.", err)
	}
	if len(toFe.History()) > 0 {
		log.Fatalf("Destination is not empty.")
	}
	list, err := fromFe.List()
	if err != nil {
		log.Fatalf("Failed to get list of files: %s.", err)
	}
	for _, h := range list {
		data, metadata, err := fromFe.Get(h.Key)
		if err != nil {
			log.Fatalf("Get(%q): %s.", h.Key, err)
		}
		if err := toFe.Put(h.Key, data, metadata); err != nil {
			log.Fatalf("Put(%q): %s.", h.Key, err)
		}
	}
	if err := toFe.Sync(); err != nil {
		log.Fatalf("Failed to sync: %s.", err)
	}
}
