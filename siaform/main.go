package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/starius/invisiblefs/siaform/files"
	"github.com/starius/invisiblefs/siaform/manager"
	"github.com/starius/invisiblefs/siaform/siaclient"
)

var (
	httpAddr   = flag.String("http-addr", "127.0.0.1:23760", "HTTP server addrer.")
	siaAddr    = flag.String("sia-addr", "127.0.0.1:9980", "Sia API addrer.")
	sectorSize = flag.Int("sector-size", 4*1024*1024, "Sia block size")
	dataDir    = flag.String("data-dir", "data-dir", "Directory to store databases")

	sc *siaclient.SiaClient
	mn *manager.Manager
	fi *files.Files
)

const indexPage = `
<html>
<body>

<form action="/upload" method="post" enctype="multipart/form-data">
    Select file to upload:
    <br>
    <input type="file" name="data" id="data">
    <br>
    <input type="submit" value="Upload File" name="submit">
</form>

</body>
</html>
`

func h(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" && req.URL.Path == "/" {
		res.Write([]byte(indexPage))
		return
	} else if req.Method == "GET" {
		parts := strings.Split(req.URL.Path, "/")
		if len(parts) < 2 {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		name := parts[1]
		f, err := fi.Open(name)
		if err != nil {
			log.Printf("fi.Open(%q): %v.", name, err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Printf("size=%d", f.File.Size)
		http.ServeContent(res, req, name, time.Time{}, f)
		return
	} else if req.Method == "POST" && req.URL.Path == "/upload" {
		log.Printf("Started uploading\n")
		name := fmt.Sprintf("%d", rand.Int())
		f, err := fi.Create(name)
		if err != nil {
			log.Printf("fi.Create(%q): %v.", name, err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		mr, err := req.MultipartReader()
		if err != nil {
			log.Printf("req.MultipartReader: %v.", err)
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("mr.NextPart: %v", err)
				return
			}
			if part.FormName() == "data" {
				for {
					buf := make([]byte, *sectorSize)
					n, err := io.ReadFull(part, buf)
					buf1 := buf
					theLast := false
					if err == io.ErrUnexpectedEOF {
						buf1 = buf[:n]
						theLast = true
					} else if err != nil {
						log.Printf("io.ReadFull: %v.", err)
						res.WriteHeader(http.StatusInternalServerError)
						return
					}
					if _, err := f.Write(buf1); err != nil {
						log.Printf("f.Write: %v.", err)
						res.WriteHeader(http.StatusInternalServerError)
						return
					}
					if theLast {
						break
					}
				}
			}
			if err := part.Close(); err != nil {
				log.Printf("part.Close: %v.", err)
				res.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		res.Write([]byte(fmt.Sprintf("<a href=/%s>Link</a>", name)))
		return
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()
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
		mn, err = manager.Load(data, sc)
		if err != nil {
			log.Fatalf("manager.Load: %v.", err)
		}
	} else {
		mn, err = manager.New(1, 0, sc)
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
	s := &http.Server{
		Addr:           *httpAddr,
		ReadTimeout:    1000 * time.Second,
		WriteTimeout:   1000 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}
	s.Handler = http.HandlerFunc(h)
	log.Fatal(s.ListenAndServe())
}
