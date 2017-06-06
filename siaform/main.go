package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

var (
	httpAddr   = flag.String("http-addr", "127.0.0.1:23760", "HTTP server addrer.")
	siaAddr    = flag.String("sia-addr", "127.0.0.1:9980", "Sia API addrer.")
	sectorSize = flag.Int("sector-size", 4*1024*1024, "Sia block size")

	client = &http.Client{}
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
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "http",
			Host:   *siaAddr,
			Path:   "/renter/contracts",
		},
		Header: map[string][]string{
			"User-Agent": {"Sia-Agent"},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("client.Do: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ioutil.ReadAll(resp.Body): %v", err)
	}
	var cj contractsJson
	if err := json.Unmarshal(body, &cj); err != nil {
		return "", fmt.Errorf("json.Unmarshal(body): %v", err)
	}
	if cj.Message != "" {
		return "", fmt.Errorf("cj.Message: %s", err)
	}
	if len(cj.Contracts) == 0 {
		return "", fmt.Errorf("no contracts")
	}
	n := rand.Intn(len(cj.Contracts))
	return cj.Contracts[n].Id, nil
}

type writeJson struct {
	SectorRoot string `json:"sector_root"`
	Message    string
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
		path2 := "/renter/read/" + contractID + "/" + sectorRoot
		req := &http.Request{
			Method: "GET",
			URL: &url.URL{
				Scheme: "http",
				Host:   *siaAddr,
				Path:   path2,
			},
			Header: map[string][]string{
				"User-Agent": {"Sia-Agent"},
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("client.Do: %v.", err)
			res.WriteHeader(http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("ioutil.ReadAll(resp.Body): %v.", err)
			res.WriteHeader(http.StatusBadGateway)
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
		body2 := &bytes.Buffer{}
		writer := multipart.NewWriter(body2)
		part, err := writer.CreateFormFile("data", "filename")
		if err != nil {
			log.Printf("writer.CreateFormFile: %v.", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if _, err := part.Write(body); err != nil {
			log.Printf("part.Write(body): %v.", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err = writer.Close(); err != nil {
			log.Printf("writer.Close: %v.", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		url := "http://" + *siaAddr + "/renter/write/" + contractID
		req, err := http.NewRequest("POST", url, body2)
		if err != nil {
			log.Printf("http.NewRequest: %v.", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		req.Header.Set("User-Agent", "Sia-Agent")
		req.Header.Set("Content-Type", writer.FormDataContentType())
		resp, err := client.Do(req)
		defer resp.Body.Close()
		body3, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("ioutil.ReadAll(resp.Body): %v.", err)
			res.WriteHeader(http.StatusBadGateway)
			return
		}
		var wr writeJson
		if err := json.Unmarshal(body3, &wr); err != nil {
			log.Printf("json.Unmarshal(body3): %v.", err)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if wr.Message != "" {
			log.Printf("wr.Message: %s.", wr.Message)
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		path3 := "/" + contractID + "/" + wr.SectorRoot + "/" + filepath.Base(fh.Filename)
		http.Redirect(res, req, path3, http.StatusFound)
		return
	}
}

func main() {
	flag.Parse()
	// Run HTTP server.
	s := &http.Server{
		Addr:           *httpAddr,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}
	s.Handler = http.HandlerFunc(h)
	log.Fatal(s.ListenAndServe())
}
