package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"os"
	"path"
)

type HAProxyGlobalConfig struct {
	Globals
	Backends             []HAProxyBackendConfig
	Certificate          string
	HTTPPort             int
	HTTPSPort            int
	ListenAddress        string
	Maxconn              int
	Nbthread             int
	OutputDir            string
	StatsPort            int
	UseUnixDomainSockets bool
}

type HAProxyBackendConfig struct {
	BackendCookie string
	ListenAddress string
	Name          string
	OutputDir     string
	Port          string
	ServerCookie  string
	TLSCACert     string
	TrafficType   TrafficType
}

const (
	HTTPBackendMapName      = "os_http_be.map"
	ReencryptBackendMapName = "os_edge_reencrypt_be.map"
	SNIPassthroughMapName   = "os_sni_passthrough.map"
	TCPBackendMapName       = "os_tcp_be.map"
	HTTPRedirectMapName     = "os_route_http_redirect.map"
)

//go:embed templates/haproxy/globals.tmpl
var globalTemplate string

//go:embed templates/haproxy/defaults.tmpl
var defaultTemplate string

//go:embed templates/haproxy/backends.tmpl
var backendTemplate string

//go:embed templates/haproxy/error-page-404.http
var error404 string

//go:embed templates/haproxy/error-page-503.http
var error503 string

func cookie() string {
	runes := []rune("0123456789abcdef")
	b := make([]rune, 32)
	for i := 0; i < 32; i++ {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}

func filterBackendsByType(types []TrafficType, backends []HAProxyBackendConfig) []HAProxyBackendConfig {
	var result []HAProxyBackendConfig

	for _, t := range types {
		for i := range backends {
			if backends[i].TrafficType == t {
				result = append(result, backends[i])
			}
		}
	}

	return result
}

func (c *GenProxyConfigCmd) Run(p *ProgramCtx) error {
	backendsByTrafficType, err := fetchAllBackendMetadata(p.DiscoveryURL)
	if err != nil {
		return err
	}

	certBundle, err := fetchCertficates(p.DiscoveryURL)
	if err != nil {
		return err
	}

	certPaths, err := writeCertificates(path.Join(p.OutputDir, "certs"), certBundle)
	if err != nil {
		return err
	}

	var proxyBackends []HAProxyBackendConfig

	for t, backends := range backendsByTrafficType {
		for _, b := range backends {
			proxyBackends = append(proxyBackends, HAProxyBackendConfig{
				BackendCookie: cookie(),
				ListenAddress: b.ListenAddress,
				Name:          b.Name,
				OutputDir:     p.OutputDir,
				Port:          fmt.Sprintf("%v", b.Port),
				ServerCookie:  cookie(),
				TLSCACert:     certPaths.RootCAFile,
				TrafficType:   t,
			})
		}
	}

	// wipe and recreate all known paths for haproxy config.
	for _, dirPath := range [][]string{
		{"haproxy"},
	} {
		paths := path.Join(p.OutputDir, path.Join(dirPath...))
		if err := os.RemoveAll(paths); err != nil {
			return err
		}
		if err := os.MkdirAll(paths, 0755); err != nil {
			return err
		}
	}

	if err := c.generateMainConfig(p, proxyBackends, certPaths.DomainFile); err != nil {
		return err
	}

	if err := c.generateMapFiles(p, proxyBackends); err != nil {
		return err
	}

	if err := c.generateCertConfig(p, proxyBackends, certPaths.DomainFile); err != nil {
		return err
	}

	return nil
}

func (c *GenProxyConfigCmd) generateMainConfig(p *ProgramCtx, backends []HAProxyBackendConfig, certFile string) error {
	config := HAProxyGlobalConfig{
		Backends:             backends,
		Certificate:          certFile,
		Globals:              p.Globals,
		HTTPPort:             c.HTTPPort,
		HTTPSPort:            c.HTTPSPort,
		ListenAddress:        c.ListenAddress,
		Maxconn:              c.Maxconn,
		Nbthread:             c.Nthreads,
		OutputDir:            p.OutputDir,
		StatsPort:            c.StatsPort,
		UseUnixDomainSockets: c.UseUnixDomainSockets,
	}

	var haproxyConf bytes.Buffer

	for _, tmpl := range []*template.Template{
		template.Must(template.New("globals").Parse(globalTemplate)),
		template.Must(template.New("defaults").Parse(defaultTemplate)),
		template.Must(template.New("backends").Parse(backendTemplate)),
	} {
		if err := tmpl.Execute(&haproxyConf, config); err != nil {
			return err
		}
	}

	if err := createFile(path.Join(p.OutputDir, "haproxy", "haproxy.config"), haproxyConf.Bytes()); err != nil {
		return err
	}

	if err := createFile(path.Join(p.OutputDir, "haproxy", "error-page-404.http"), bytes.NewBuffer([]byte(error404)).Bytes()); err != nil {
		return err
	}

	return createFile(path.Join(p.OutputDir, "haproxy", "error-page-503.http"), bytes.NewBuffer([]byte(error503)).Bytes())
}

func (c *GenProxyConfigCmd) generateMapFiles(p *ProgramCtx, backends []HAProxyBackendConfig) error {
	type MapEntryFunc func(backend HAProxyBackendConfig) string

	backendMaps := []struct {
		MapName      string
		TrafficTypes []TrafficType
		MapEntry     MapEntryFunc
		Buffer       *bytes.Buffer
	}{{
		MapName:      HTTPBackendMapName,
		TrafficTypes: []TrafficType{HTTPTraffic},
		Buffer:       &bytes.Buffer{},
		MapEntry: func(b HAProxyBackendConfig) string {
			switch b.TrafficType {
			case HTTPTraffic:
				return fmt.Sprintf("^%s\\.?(:[0-9]+)?(/.*)?$ be_http:%s\n", b.Name, b.Name)
			default:
				panic("unexpected traffic type: " + b.TrafficType)
			}
		},
	}, {
		MapName:      ReencryptBackendMapName,
		TrafficTypes: []TrafficType{ReencryptTraffic, EdgeTraffic},
		Buffer:       &bytes.Buffer{},
		MapEntry: func(b HAProxyBackendConfig) string {
			switch b.TrafficType {
			case EdgeTraffic:
				return fmt.Sprintf("^%s\\.?(:[0-9]+)?(/.*)?$ be_edge_http:%s\n", b.Name, b.Name)
			case ReencryptTraffic:
				return fmt.Sprintf("^%s\\.?(:[0-9]+)?(/.*)?$ be_secure:%s\n", b.Name, b.Name)
			default:
				panic("unexpected traffic type: " + b.TrafficType)
			}
		},
	}, {
		MapName:      SNIPassthroughMapName,
		TrafficTypes: []TrafficType{PassthroughTraffic},
		Buffer:       &bytes.Buffer{},
		MapEntry: func(b HAProxyBackendConfig) string {
			switch b.TrafficType {
			case PassthroughTraffic:
				return fmt.Sprintf("^%s$ 1\n", b.Name)
			default:
				panic("unexpected traffic type: " + b.TrafficType)
			}
		},
	}, {
		MapName:      TCPBackendMapName,
		TrafficTypes: []TrafficType{PassthroughTraffic},
		Buffer:       &bytes.Buffer{},
		MapEntry: func(b HAProxyBackendConfig) string {
			switch b.TrafficType {
			case PassthroughTraffic:
				return fmt.Sprintf("^%s\\.?(:[0-9]+)?(/.*)?$ be_tcp:%s\n", b.Name, b.Name)
			default:
				panic("unexpected traffic type: " + b.TrafficType)
			}
		},
	}, {
		MapName:      HTTPRedirectMapName,
		TrafficTypes: []TrafficType{},
		Buffer:       &bytes.Buffer{},
		MapEntry: func(b HAProxyBackendConfig) string {
			// no support for redirects; this is deliberate
			return ""
		},
	}}

	for _, m := range backendMaps {
		for _, b := range filterBackendsByType(m.TrafficTypes, backends) {
			if _, err := io.WriteString(m.Buffer, m.MapEntry(b)); err != nil {
				return err
			}
		}
		if err := createFile(path.Join(p.OutputDir, "haproxy", m.MapName), m.Buffer.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func (c *GenProxyConfigCmd) generateCertConfig(p *ProgramCtx, backends []HAProxyBackendConfig, certFile string) error {
	var certConfigMap bytes.Buffer

	for _, b := range filterBackendsByType([]TrafficType{EdgeTraffic, ReencryptTraffic}, backends) {
		if _, err := io.WriteString(&certConfigMap, fmt.Sprintf("%s %s\n", certFile, b.Name)); err != nil {
			return err
		}
	}

	return createFile(path.Join(p.OutputDir, "haproxy", "cert_config.map"), certConfigMap.Bytes())
}
