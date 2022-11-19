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

type portSelector func(b BoundBackend, cfg Globals) int
type schemeSelector func(t TrafficType) string

func haproxyPortSelector(b BoundBackend, cfg Globals) int {
	switch b.TrafficType {
	case HTTPTraffic:
		return cfg.HTTPPort
	default:
		return cfg.HTTPSPort
	}
}

func haproxySNIOnlyPortSelector(b BoundBackend, cfg Globals) int {
	switch b.TrafficType {
	case HTTPTraffic:
		return cfg.HTTPPort
	default:
		return cfg.HTTPSPortSNIOnly
	}
}

func haproxySchemeSelector(t TrafficType) string {
	switch t {
	case HTTPTraffic:
		return "http"
	default:
		return "https"
	}
}

func directPortSelector(b BoundBackend, cfg Globals) int {
	return b.Port
}

func directSchemeSelector(t TrafficType) string {
	switch t {
	case HTTPTraffic, EdgeTraffic:
		return "http"
	default:
		return "https"
	}
}

func generateMBRequests(p *ProgramCtx, portSelector portSelector, schemeSelector schemeSelector, cfg MBRequestConfig, backends []BoundBackend) []MBRequest {
	var requests []MBRequest

	for _, b := range backends {
		requests = append(requests, MBRequest{
			Clients:           cfg.Clients,
			Host:              b.Name,
			KeepAliveRequests: cfg.KeepAliveRequests,
			Method:            "GET",
			Path:              "/1024.html",
			Port:              portSelector(b, p.Globals),
			Scheme:            schemeSelector(b.TrafficType),
			TLSSessionReuse:   cfg.TLSSessionReuse,
		})
	}

	return requests
}

func (c *GenWorkloadCmd) Run(p *ProgramCtx) error {
	basedir := path.Join(p.OutputDir, "requests")
	if err := os.RemoveAll(basedir); err != nil {
		return err
	}

	backendsByTrafficType, err := fetchAllBackendMetadata(p.DiscoveryURL)
	if err != nil {
		return err
	}

	for _, workload := range []struct {
		subdir         string
		useProxy       bool
		portSelector   portSelector
		schemeSelector schemeSelector
	}{
		{"direct", false, directPortSelector, directSchemeSelector},
		{"haproxy", true, haproxyPortSelector, haproxySchemeSelector},
		{"haproxy-reencrypt-only", true, haproxySNIOnlyPortSelector, haproxySchemeSelector},
	} {
		for _, clients := range []int{1, 2, 5, 10, 50, 75, 80, 90, 100, 200} {
			for _, requestCfg := range []struct {
				Name         string
				TrafficTypes []TrafficType
			}{
				{"edge", []TrafficType{EdgeTraffic}},
				{"http", []TrafficType{HTTPTraffic}},
				{"mix", AllTrafficTypes[:]},
				{"passthrough", []TrafficType{PassthroughTraffic}},
				{"reencrypt", []TrafficType{ReencryptTraffic}},
			} {
				if workload.subdir == "haproxy-reencrypt-only" && requestCfg.Name != "reencrypt" {
					continue
				}
				for _, keepAliveRequests := range []int{0} {
					config := MBRequestConfig{
						Clients:           clients,
						KeepAliveRequests: keepAliveRequests,
						TLSSessionReuse:   p.TLSReuse,
						TrafficTypes:      requestCfg.TrafficTypes,
					}
					requests := generateMBRequests(p, workload.portSelector, workload.schemeSelector, config, filterInTrafficByType(requestCfg.TrafficTypes, backendsByTrafficType))
					data, err := json.MarshalIndent(requests, "", "  ")
					if err != nil {
						return err
					}
					filepath := fmt.Sprintf("%s/%s/traffic-%v-backends-%v-clients-%v-keepalives-%v.json",
						basedir,
						workload.subdir,
						requestCfg.Name,
						len(requests),
						config.Clients,
						config.KeepAliveRequests)
					if err := createFile(filepath, data); err != nil {
						return fmt.Errorf("error generating %s: %v", filepath, err)
					}
				}
			}
		}
	}

	return nil
}
