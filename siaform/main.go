package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/starius/invisiblefs/siaform/siaclient"
)

var (
	httpAddr   = flag.String("http-addr", "127.0.0.1:23760", "HTTP server addrer.")
	siaAddr    = flag.String("sia-addr", "127.0.0.1:9980", "Sia API addrer.")
	sectorSize = flag.Int("sector-size", 4*1024*1024, "Sia block size")

	sc *siaclient.SiaClient
)

const indexPage = `
<html>
<body>

<form action="/upload" method="post" enctype="multipart/form-data">
    Select file to upload (max %d bytes):
    <br>
    <input type="file" name="data" id="data">
    <br>
    <input type="submit" value="Upload File" name="submit">
</form>

</body>
</html>
`

type contractsJson struct {
	Contracts []struct {
		Id string
	}
	Message string
}

func selectContract() (string, error) {
	contracts, err := sc.Contracts()
	if err != nil {
		return "", err
	}
	if len(contracts) == 0 {
		return "", fmt.Errorf("no contracts")
	}
	n := rand.Intn(len(contracts))
	return contracts[n], nil
}

func h(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" && req.URL.Path == "/" {
		res.Write([]byte(fmt.Sprintf(indexPage, *sectorSize)))
		return
	} else if req.Method == "GET" {
		parts := strings.Split(req.URL.Path, "/")
		if len(parts) < 3 {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		contractID := parts[1]
		sectorRoot := parts[2]
		body, err := sc.Read(contractID, sectorRoot)
		if err != nil {
			log.Printf("sc.Read(%q, %q): %v.", contractID, sectorRoot, err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(body) < 4 {
			log.Printf("len(body): %d.", len(body))
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		length := binary.LittleEndian.Uint32(body[:4])
		if len(body) < int(4+length) {
			log.Printf("len(body): %d. length: %d.", len(body), length)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		data := body[4 : 4+length]
		res.Write(data)
		return
	} else if req.Method == "POST" && req.URL.Path == "/upload" {
		f, fh, err := req.FormFile("data")
		if err != nil {
			log.Printf("req.FormFile: %v.", err)
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		data, err := ioutil.ReadAll(f)
		if err != nil {
			log.Printf("ioutil.ReadAll(f): %v.", err)
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(data)+4 > *sectorSize {
			res.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		body := make([]byte, 4)
		binary.LittleEndian.PutUint32(body, uint32(len(data)))
		body = append(body, data...)
		body = append(body, make([]byte, *sectorSize-len(body))...)
		contractID, err := selectContract()
		if err != nil {
			log.Printf("selectContract: %v.", err)
			res.WriteHeader(http.StatusBadGateway)
			return
		}
		sectorRoot, err := sc.Write(contractID, body)
		if err != nil {
			log.Printf("sc.Write(%q, body): %v.", contractID, err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		path3 := "/" + contractID + "/" + sectorRoot + "/" + filepath.Base(fh.Filename)
		http.Redirect(res, req, path3, http.StatusFound)
		return
	}
}

func main() {
	flag.Parse()
	var err error
	sc, err = siaclient.New(*siaAddr, &http.Client{})
	if err != nil {
		log.Fatalf("siaclient.New: %v.", err)
	}
	s := &http.Server{
		Addr:           *httpAddr,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}
	s.Handler = http.HandlerFunc(h)
	log.Fatal(s.ListenAndServe())
}
