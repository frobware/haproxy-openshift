package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

type Request struct {
	Clients           int64  `json:"clients"`
	Delay             Delay  `json:"delay"`
	Host              string `json:"host"`
	KeepAliveRequests int64  `json:"keep-alive-requests"`
	Method            string `json:"method"`
	Path              string `json:"path"`
	Port              int64  `json:"port"`
	Scheme            string `json:"scheme"`
	TLSSessionReuse   bool   `json:"tls-session-reuse"`
}

type Delay struct {
	Max int64 `json:"max"`
	Min int64 `json:"min"`
}

func generateWorkloadRequests(cfg RequestConfig, backends []BoundBackend) []Request {
	var requests []Request

	for _, b := range backends {
		requests = append(requests, Request{
			Clients:           cfg.Clients,
			Host:              fmt.Sprintf("%s", b.Name),
			KeepAliveRequests: cfg.KeepAliveRequests,
			Method:            "GET",
			Path:              "/1024.html",
			Port:              b.TrafficType.Port(),
			Scheme:            b.TrafficType.Scheme(),
			TLSSessionReuse:   cfg.TLSSessionReuse,
		})
	}

	return requests
}

func filterInTrafficByType(types []TrafficType, backendsMap BoundBackendsByTrafficType) []BoundBackend {
	var result []BoundBackend

	for _, t := range types {
		result = append(result, backendsMap[t]...)
	}

	return result
}

func (c *GenWorkloadCmd) Run(p *ProgramCtx) error {
	backends, err := fetchAllBackendMetadata(p.DiscoveryURL)
	if err != nil {
		return err
	}

	for _, clients := range []int64{1, 10, 50, 100, 200} {
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
			for _, keepAliveRequests := range []int64{0} {
				config := RequestConfig{
					Clients:           clients,
					KeepAliveRequests: keepAliveRequests,
					TLSSessionReuse:   c.TLSReuse,
					TrafficTypes:      scenario.TrafficTypes,
				}
				requests := generateWorkloadRequests(config, filterInTrafficByType(scenario.TrafficTypes, backends))
				data, err := json.MarshalIndent(requests, "", "  ")
				if err != nil {
					return err
				}
				path := fmt.Sprintf("%s/mb/traffic-%v-backends-%v-clients-%v-keepalives-%v",
					p.OutputDir,
					scenario.Name,
					len(requests)/len(config.TrafficTypes),
					config.Clients,
					config.KeepAliveRequests)
				if err := os.MkdirAll(path, 0755); err != nil {
					return fmt.Errorf("failed to create path: %q: %v", path, err)
				}
				filename := fmt.Sprintf("%s/requests.json", path)
				if err := createFile(filename, data); err != nil {
					return fmt.Errorf("error generating %s: %v", filename, err)
				}
			}
		}
	}

	return nil
}
