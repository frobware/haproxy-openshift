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

func (c *GenProxyConfigCmd) generateMBRequests(cfg MBRequestConfig, backends []BoundBackend) []MBRequest {
	var requests []MBRequest

	port := func(t TrafficType) int {
		switch t {
		case HTTPTraffic:
			return c.HTTPPort
		default:
			return c.HTTPSPort
		}
	}

	scheme := func(t TrafficType) string {
		switch t {
		case HTTPTraffic:
			return "http"
		default:
			return "https"
		}
	}

	for _, b := range backends {
		requests = append(requests, MBRequest{
			Clients:           cfg.Clients,
			Host:              b.Name,
			KeepAliveRequests: cfg.KeepAliveRequests,
			Method:            "GET",
			Path:              "/1024.html",
			Port:              port(b.TrafficType),
			Scheme:            scheme(b.TrafficType),
			TLSSessionReuse:   cfg.TLSSessionReuse,
		})
	}

	return requests
}

func (c *GenProxyConfigCmd) generateMBTestScenarios(p *ProgramCtx) error {
	if err := os.RemoveAll(path.Join(p.OutputDir, "mb")); err != nil {
		return err
	}

	backendsByTrafficType, err := fetchAllBackendMetadata(p.DiscoveryURL)
	if err != nil {
		return err
	}

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
				TLSSessionReuse:   c.TLSReuse,
				TrafficTypes:      scenario.TrafficTypes,
			}
			requests := c.generateMBRequests(config, filterInTrafficByType(scenario.TrafficTypes, backendsByTrafficType))
			data, err := json.MarshalIndent(requests, "", "  ")
			if err != nil {
				return err
			}
			filepath := fmt.Sprintf("%s/mb/traffic-%v-backends-%v-clients-%v-keepalives-%v-requests.json",
				p.OutputDir,
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
