package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
)

// https://github.com/jmencak/mb

// MBRequest is a Multiple-host HTTP(s) Benchmarking tool request.
type MBRequest struct {
	Clients           int    `json:"clients"`
	Host              string `json:"host"`
	KeepAliveRequests int    `json:"keep-alive-requests"`
	Method            string `json:"method"`
	Path              string `json:"path"`
	Port              int    `json:"port"`
	Scheme            string `json:"scheme"`
	TLSSessionReuse   bool   `json:"tls-session-reuse"`
}

type MBRequestConfig struct {
	Clients           int
	KeepAliveRequests int
	TLSSessionReuse   bool
	TrafficTypes      []TrafficType
}

func filterInTrafficByType(types []TrafficType, backendsMap BoundBackendsByTrafficType) []BoundBackend {
	var result []BoundBackend

	for _, t := range types {
		result = append(result, backendsMap[t]...)
	}

	return result
}

func generateMBRequests(p *ProgramCtx, useProxy bool, cfg MBRequestConfig, backends []BoundBackend) []MBRequest {
	var requests []MBRequest

	port := func(b BoundBackend, useProxy bool) int {
		switch useProxy {
		case true:
			switch b.TrafficType {
			case HTTPTraffic:
				return p.HTTPPort
			default:
				return p.HTTPSPort
			}
		default:
			return b.Port
		}
	}

	scheme := func(t TrafficType, useProxy bool) string {
		switch useProxy {
		case true:
			switch t {
			case HTTPTraffic:
				return "http"
			default:
				return "https"
			}
		default:
			switch t {
			case HTTPTraffic, EdgeTraffic:
				return "http"
			default:
				return "https"
			}
		}
	}

	for _, b := range backends {
		requests = append(requests, MBRequest{
			Clients:           cfg.Clients,
			Host:              b.Name,
			KeepAliveRequests: cfg.KeepAliveRequests,
			Method:            "GET",
			Path:              "/1024.html",
			Port:              port(b, useProxy),
			Scheme:            scheme(b.TrafficType, useProxy),
			TLSSessionReuse:   cfg.TLSSessionReuse,
		})
	}

	return requests
}

func (c *GenWorkloadCmd) writeRequests(p *ProgramCtx, category string, useProxy bool, backendsByTrafficType BoundBackendsByTrafficType) error {
	for _, scenario := range []struct {
		Name         string
		TrafficTypes []TrafficType
	}{
		{"edge", []TrafficType{EdgeTraffic}},
		{"http", []TrafficType{HTTPTraffic}},
		{"mix", AllTrafficTypes[:]},
		{"passthrough", []TrafficType{PassthroughTraffic}},
		{"reencrypt", []TrafficType{ReencryptTraffic}},
	} {
		for _, keepAliveRequests := range []int{0} {
			config := MBRequestConfig{
				Clients:           len(backendsByTrafficType[EdgeTraffic]),
				KeepAliveRequests: keepAliveRequests,
				TLSSessionReuse:   p.TLSReuse,
				TrafficTypes:      scenario.TrafficTypes,
			}
			requests := generateMBRequests(p, useProxy, config, filterInTrafficByType(scenario.TrafficTypes, backendsByTrafficType))
			data, err := json.MarshalIndent(requests, "", "  ")
			if err != nil {
				return err
			}
			filepath := fmt.Sprintf("%s/mb/%s/traffic-%v-backends-%v-clients-%v-keepalives-%v-requests.json",
				p.OutputDir,
				category,
				scenario.Name,
				len(requests)/len(config.TrafficTypes),
				config.Clients,
				config.KeepAliveRequests)
			if err := createFile(filepath, data); err != nil {
				return fmt.Errorf("error generating %s: %v", filepath, err)
			}
		}
	}

	return nil
}

func (c *GenWorkloadCmd) Run(p *ProgramCtx) error {
	if err := os.RemoveAll(path.Join(p.OutputDir, "mb")); err != nil {
		return err
	}

	backendsByTrafficType, err := fetchAllBackendMetadata(p.DiscoveryURL)
	if err != nil {
		return err
	}

	for _, cfg := range []struct {
		subdir   string
		useProxy bool
	}{
		{"direct", false},
		{"haproxy", true},
	} {
		if err := c.writeRequests(p, cfg.subdir, cfg.useProxy, backendsByTrafficType); err != nil {
			return err
		}
	}

	return nil
}
