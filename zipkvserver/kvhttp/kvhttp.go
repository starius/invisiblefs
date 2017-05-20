package kvhttp

import (
	"io"
	"log"
	"net/http"

	"github.com/starius/invisiblefs/zipkvserver/zipkv"
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

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path
	if r.Method == "GET" {
		value, err := h.kv.Get(key)
		if err != nil {
			log.Printf("Get(%q): %s", key, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(value); err != nil {
			log.Printf("Write: %s", err)
			return
		}
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
