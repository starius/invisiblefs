package kvhttp

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/NebulousLabs/fastrand"
	"github.com/starius/invisiblefs/zipkvserver/zipkv"
)

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
)

type Handler struct {
	kv       zipkv.KV
	maxValue int64
}

func New(kv zipkv.KV, maxValue int) (*Handler, error) {
	return &Handler{
		kv:       kv,
		maxValue: int64(maxValue),
	}, nil
}

func genRequestId() string {
	return hex.EncodeToString(fastrand.Bytes(10))
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path
	if r.Method == "GET" {
		value, err := h.kv.Get(key)
		if err != nil {
			log.Printf("Get(%q): %s", key, err)
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf(xml404, key, genRequestId())))
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(value); err != nil {
			log.Printf("Write: %s", err)
			return
		}
	} else if r.Method == "HEAD" {
		has, err := h.kv.Has(key)
		if err != nil {
			log.Printf("Has(%q): %s", key, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if has {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
		return
	} else if r.Method == "PUT" {
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
		if err := h.kv.Put(key, value); err != nil {
			log.Printf("Put(%q): %s", key, err)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		digest := md5.Sum(value)
		w.Header().Set("ETag", hex.EncodeToString(digest[:]))
		w.WriteHeader(http.StatusOK)
	} else if r.Method == "DELETE" {
		if err := h.kv.Delete(key); err != nil {
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
