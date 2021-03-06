package kvhttp

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/NebulousLabs/fastrand"
	"github.com/golang/protobuf/proto"
	"github.com/starius/invisiblefs/zipkvserver/kv"
)

//go:generate protoc --proto_path=. --go_out=. metadata.proto

// http://docs.aws.amazon.com/AmazonS3/latest/API/Welcome.html

const (
	xml404 = `<?xml version="1.0" encoding="UTF-8"?>
<Error>
  <Code>NoSuchKey</Code>
  <Message>The resource you requested does not exist</Message>
  <Resource>%s</Resource>
  <RequestId>%s</RequestId>
</Error>
	`
	xmlCopy = `<?xml version="1.0" encoding="UTF-8"?>
<CopyObjectResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
</CopyObjectResult>
	`
	xmlListHead = `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
    <Name>bucket</Name>
    <Prefix/>
    <KeyCount>%d</KeyCount>
    <MaxKeys>%d</MaxKeys>
    <IsTruncated>false</IsTruncated>
	`
	xmlListContents = `
    <Contents>
        <Key>%s</Key>
        <Size>%d</Size>
        <StorageClass>STANDARD</StorageClass>
    </Contents>
	`
	xmlListTail = `
</ListBucketResult>
	`
)

type Handler struct {
	kv       kv.KV
	baseURL  string
	maxValue int64
}

func New(kv kv.KV, maxValue int, baseURL string) (*Handler, error) {
	return &Handler{
		kv:       kv,
		baseURL:  baseURL,
		maxValue: int64(maxValue),
	}, nil
}

func genRequestId() string {
	return hex.EncodeToString(fastrand.Bytes(10))
}

func writeMetadata(w http.ResponseWriter, metadata []byte) error {
	if metadata == nil {
		return nil
	}
	md := &Metadata{}
	if err := proto.Unmarshal(metadata, md); err != nil {
		return fmt.Errorf("proto.Unmarshal: %s", err)
	}
	for mdKey, mdValue := range md.Metadata {
		w.Header().Set(mdKey, mdValue)
	}
	return nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path
	log.Printf("%s %s?%s", r.Method, key, r.URL.RawQuery)
	if !strings.HasPrefix(key, h.baseURL) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	key = strings.TrimPrefix(key, h.baseURL)
	if r.Method == "GET" && key == "" {
		prefix := r.URL.Query().Get("prefix")
		list, err := h.kv.List()
		if err != nil {
			log.Printf("List(): %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		list2 := make(map[string]int)
		for key, size := range list {
			if strings.HasPrefix(key, prefix) {
				list2[key] = size
			}
		}
		var buf []string
		l := len(list2)
		buf = append(buf, fmt.Sprintf(xmlListHead, l, l))
		for key, size := range list {
			if strings.HasPrefix(key, prefix) {
				cs := fmt.Sprintf(xmlListContents, key, size)
				buf = append(buf, cs)
			}
		}
		buf = append(buf, xmlListTail)
		buf2 := strings.Join(buf, "")
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(buf2)); err != nil {
			log.Printf("Write: %s", err)
			return
		}
		return
	} else if r.Method == "GET" || r.Method == "HEAD" {
		// TODO Range: bytes.
		value, metadata, err := h.kv.Get(key)
		if err != nil {
			log.Printf("Get(%q): %s", key, err)
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf(xml404, key, genRequestId())))
			return
		}
		if err := writeMetadata(w, metadata); err != nil {
			log.Printf("writeMetadata: %s.", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		digest := md5.Sum(value)
		w.Header().Set("ETag", hex.EncodeToString(digest[:]))
		w.WriteHeader(http.StatusOK)
		if r.Method == "GET" {
			if _, err := w.Write(value); err != nil {
				log.Printf("Write: %s", err)
				return
			}
		}
	} else if r.Method == "PUT" {
		md := &Metadata{
			Metadata: make(map[string]string),
		}
		for hrKey, hrValues := range r.Header {
			hrKeyLower := strings.ToLower(hrKey)
			if strings.HasPrefix(hrKeyLower, "x-amz-meta-") {
				md.Metadata[hrKey] = hrValues[0]
			}
		}
		var metadata []byte
		if len(md.Metadata) > 0 {
			var err error
			metadata, err = proto.Marshal(md)
			if err != nil {
				log.Printf("proto.Marshal(md): %s.", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		copySource := r.Header.Get("x-amz-copy-source")
		if copySource != "" {
			// http://docs.aws.amazon.com/AmazonS3/latest/API/RESTObjectCOPY.html
			copySource := strings.TrimPrefix(copySource, h.baseURL)
			log.Printf("x-amz-copy-source: %s", copySource)
			has, srcMd, err := h.kv.Has(copySource)
			if err != nil {
				log.Printf("h.kv.Has(%q): %s", copySource, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if !has {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(fmt.Sprintf(xml404, copySource, genRequestId())))
				return
			}
			mdDir := r.Header.Get("x-amz-metadata-directive")
			if mdDir == "COPY" || mdDir == "" {
				metadata = srcMd
			} else if mdDir == "REPLACE" {
				// Nothing to do, metadata has been read above.
			} else {
				log.Printf("bad x-amz-metadata-directive: %s.", mdDir)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if err := h.kv.Link(key, copySource, metadata); err != nil {
				log.Printf("h.kv.Has(%q): %s", copySource, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(xmlCopy))
			return
		}
		if r.ContentLength > h.maxValue {
			log.Printf("%d > %d", r.ContentLength, h.maxValue)
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		value := make([]byte, r.ContentLength)
		if _, err := io.ReadFull(r.Body, value); err != nil {
			log.Printf("io.ReadFull: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := h.kv.Put(key, value, metadata); err != nil {
			log.Printf("Put(%q): %s", key, err)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		digest := md5.Sum(value)
		w.Header().Set("ETag", hex.EncodeToString(digest[:]))
		w.WriteHeader(http.StatusOK)
		return
	} else if r.Method == "DELETE" {
		if _, err := h.kv.Delete(key); err != nil {
			log.Printf("Delete(%q): %s", key, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}
