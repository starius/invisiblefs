package gzip

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

func Gzip(data []byte) ([]byte, error) {
	var b bytes.Buffer
	gz, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Flush(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func Gunzip(zdata []byte) ([]byte, error) {
	b := bytes.NewBuffer(zdata)
	gz, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(gz)
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return data, err
}
