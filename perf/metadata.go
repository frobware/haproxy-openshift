package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func fetchAllBackendMetadata(uri string) (BoundBackendsByTrafficType, error) {
	url := fmt.Sprintf("%s/backends?json=1", uri)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) { _ = Body.Close() }(resp.Body)
	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var backendsByType BoundBackendsByTrafficType
		if err := json.Unmarshal(body, &backendsByType); err != nil {
			return nil, err
		}
		return backendsByType, nil
	}
	return nil, fmt.Errorf("/backends request failed %v", resp.StatusCode)
}

func fetchCertficates(uri string) (*CertificateBundle, error) {
	url := fmt.Sprintf("%s/certs", uri)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) { _ = Body.Close() }(resp.Body)
	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var certBundle CertificateBundle
		if err := json.Unmarshal(body, &certBundle); err != nil {
			return nil, err
		}
		return &certBundle, nil
	}
	return nil, fmt.Errorf("/certs request failed %v", resp.StatusCode)
}
