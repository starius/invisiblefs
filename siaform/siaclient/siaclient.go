package siaclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
)

type SiaClient struct {
	siaAddr string

	client *http.Client
}

func New(siaAddr string, client *http.Client) (*SiaClient, error) {
	return &SiaClient{
		siaAddr: siaAddr,
		client:  client,
	}, nil
}

type contractsJson struct {
	Contracts []struct {
		Id string
	}
	Message string
}

func (s *SiaClient) Contracts() ([]string, error) {
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "http",
			Host:   s.siaAddr,
			Path:   "/renter/contracts",
		},
		Header: map[string][]string{
			"User-Agent": {"Sia-Agent"},
		},
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client.Do: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll(resp.Body): %v", err)
	}
	var cj contractsJson
	if err := json.Unmarshal(body, &cj); err != nil {
		return nil, fmt.Errorf("json.Unmarshal(body): %v", err)
	}
	if cj.Message != "" {
		return nil, fmt.Errorf("cj.Message: %s", err)
	}
	var contracts []string
	for _, c := range cj.Contracts {
		contracts = append(contracts, c.Id)
	}
	return contracts, nil
}

func (s *SiaClient) Read(contractID, sectorRoot string) ([]byte, error) {
	path2 := "/renter/read/" + contractID + "/" + sectorRoot
	req := &http.Request{
		Method: "GET",
		URL: &url.URL{
			Scheme: "http",
			Host:   s.siaAddr,
			Path:   path2,
		},
		Header: map[string][]string{
			"User-Agent": {"Sia-Agent"},
		},
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client.Do: %v.", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadAll(resp.Body): %v.", err)
	}
	return body, nil
}

type writeJson struct {
	SectorRoot string `json:"sector_root"`
	Message    string
}

func (s *SiaClient) Write(contractID string, data []byte) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("data", "filename")
	if err != nil {
		return "", fmt.Errorf("writer.CreateFormFile: %v.", err)
	}
	if _, err := part.Write(data); err != nil {
		return "", fmt.Errorf("part.Write(body): %v.", err)
	}
	if err = writer.Close(); err != nil {
		return "", fmt.Errorf("writer.Close: %v.", err)
	}
	url := "http://" + s.siaAddr + "/renter/write/" + contractID
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("http.NewRequest: %v.", err)
	}
	req.Header.Set("User-Agent", "Sia-Agent")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := s.client.Do(req)
	defer resp.Body.Close()
	body2, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ioutil.ReadAll(resp.Body): %v.", err)
	}
	var wr writeJson
	if err := json.Unmarshal(body2, &wr); err != nil {
		return "", fmt.Errorf("json.Unmarshal(body2): %v.", err)
	}
	if wr.Message != "" {
		return "", fmt.Errorf("wr.Message: %s.", wr.Message)
	}
	return wr.SectorRoot, nil
}
