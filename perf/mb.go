package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
)

func (c *GenWorkloadCmd) generateWorkloadRequests(cfg RequestConfig, backends []BoundBackend) []MBRequest {
	var requests []MBRequest

	for _, b := range backends {
		requests = append(requests, MBRequest{
			Clients:           cfg.Clients,
			Host:              b.Name,
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
		for _, keepAliveRequests := range []int64{0} {
			config := RequestConfig{
				Clients:           int64(len(backendsByTrafficType[EdgeTraffic])),
				KeepAliveRequests: keepAliveRequests,
				TLSSessionReuse:   c.TLSReuse,
				TrafficTypes:      scenario.TrafficTypes,
			}
			requests := c.generateWorkloadRequests(config, filterInTrafficByType(scenario.TrafficTypes, backendsByTrafficType))
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
