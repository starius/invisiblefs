package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/starius/invisiblefs/devindir"
)

var (
	dir        = flag.String("dir", "", "Dir with underlying files")
	mountpoint = flag.String("mountpoint", "", "Where to mount")
	bs         = flag.Int("bs", 40*1024*1024, "Block size, bytes")
	size       = flag.Int64("size", 1000000000000, "File size")
	fname      = flag.String("fname", "dev", "File name")
	cacheSize  = flag.Int("files-cache", 100, "Open files cache size")
)

func main() {
	flag.Parse()
	if *dir == "" || *mountpoint == "" {
		flag.PrintDefaults()
		log.Fatal("Provide -dir and -mountpoint")
	}
	dfs, err := devindir.New(*dir, *fname, *bs, *size, *cacheSize)
	if err != nil {
		log.Fatalf("Failed to create fs object: %s.", err)
	}
	mp, err := fuse.Mount(
		*mountpoint, fuse.FSName("devindir"),
		fuse.Subtype("devindir"), fuse.LocalVolume(),
	)
	if err != nil {
		log.Fatalf("Failed to mount FUSE: %s.", err)
	}
	defer mp.Close()
	// Handle signals - close file.
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		defer wg.Done()
		for signal := range c {
			fmt.Printf("Caught %s.\n", signal)
			fmt.Printf("Unmounting %s.\n", *mountpoint)
			if err := fuse.Unmount(*mountpoint); err != nil {
				fmt.Printf("Failed to unmount: %s.\n", err)
				continue
			}
			fmt.Printf("Successfully unmount %s.\n", *mountpoint)
			fmt.Printf("Writting remaining files to %s.\n", *dir)
			if err := dfs.Close(); err != nil {
				fmt.Printf("Failed to write: %s.\n", err)
				continue
			}
			fmt.Printf("Successfully written files.\n")
			fmt.Printf("Exiting.\n")
			return
		}
	}()
	fmt.Printf("Serving %s from %s...\n", *dir, *mountpoint)
	err = fs.Serve(mp, dfs)
	if err != nil {
		log.Fatal(err)
	}
	// check if the mount process has an error to report
	<-mp.Ready
	if err := mp.MountError; err != nil {
		log.Fatal(err)
	}
	wg.Wait()
}
