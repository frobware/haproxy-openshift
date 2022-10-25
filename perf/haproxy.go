package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
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

type Backends map[string]Backend

type RequestConfig struct {
	Clients           int64
	KeepAliveRequests int64
	TLSSessionReuse   bool
	TrafficTypes      []TrafficType
}

type HAProxyGlobalConfig struct {
	Globals

	Backends  []HAProxyBackendConfig
	HTTPPort  int
	HTTPSPort int
	Maxconn   int
	Nbthread  int
	OutputDir string
	StatsPort int
}

type HAProxyBackendConfig struct {
	BackendCookie string
	HostAddr      string
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

//go:embed haproxy/globals.tmpl
var globalTemplate string

//go:embed haproxy/defaults.tmpl
var defaultTemplate string

//go:embed haproxy/backends.tmpl
var backendTemplate string

//go:embed haproxy/error-page-404.http
var error404 string

//go:embed haproxy/error-page-503.http
var error503 string

func cookie() string {
	runes := []rune("0123456789abcdef")
	b := make([]rune, 32)
	for i := 0; i < 32; i++ {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}

func fetchBackendMetadata[T TrafficType](uri string, t T) ([]string, error) {
	url := fmt.Sprintf("%s/backends/%v", uri, t)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return strings.Split(strings.Trim(string(body), "\n"), "\n"), nil
	}

	return nil, fmt.Errorf("unexpected status %v", resp.StatusCode)
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

func generateMBRequests(cfg RequestConfig, backends []HAProxyBackendConfig) []Request {
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

func (c *GenProxyConfigCmd) Run(p *ProgramCtx) error {
	if err := os.RemoveAll(p.OutputDir); err != nil {
		return err
	}

	var backends []HAProxyBackendConfig

	for _, t := range AllTrafficTypes {
		metadata, err := fetchBackendMetadata(p.DiscoveryURL, t)
		if err != nil {
			return err
		}
		for i := range metadata {
			words := strings.Split(metadata[i], " ")
			if len(words) < 3 {
				return fmt.Errorf("not enough words in %q", metadata[i])
			}
			addrs, err := net.LookupIP(words[0])
			if err != nil {
				return err
			}
			backends = append(backends, HAProxyBackendConfig{
				BackendCookie: cookie(),
				HostAddr:      addrs[0].String(),
				Name:          words[1],
				OutputDir:     p.OutputDir,
				Port:          words[2],
				ServerCookie:  cookie(),
				TLSCACert:     p.TLSCACert,
				TrafficType:   t,
			})
		}
	}

	if err := os.MkdirAll(path.Join(p.OutputDir, "run"), 0755); err != nil {
		return err
	}

	// create known paths that need to exist.
	for _, dirPath := range [][]string{
		{"conf"},
		{"log"},
		{"router", "cacerts"},
		{"router", "certs"},
		{"run"},
	} {
		paths := path.Join(p.OutputDir, path.Join(dirPath...))
		if err := os.MkdirAll(paths, 0755); err != nil {
			return err
		}
	}

	if err := c.generateMainConfig(p, backends); err != nil {
		return err
	}

	if err := c.generateMapFiles(p, backends); err != nil {
		return err
	}

	if err := c.generateCertConfig(p, backends); err != nil {
		return err
	}

	return c.generateMBRequests(p, backends)
}

func (c *GenProxyConfigCmd) generateMainConfig(p *ProgramCtx, backends []HAProxyBackendConfig) error {
	config := HAProxyGlobalConfig{
		Globals:   p.Globals,
		Backends:  backends,
		HTTPPort:  c.HTTPPort,
		HTTPSPort: c.HTTPSPort,
		Maxconn:   c.Maxconn,
		Nbthread:  c.Nthreads,
		OutputDir: p.OutputDir,
		StatsPort: c.StatsPort,
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

	if err := createFile(path.Join(p.OutputDir, "conf", "haproxy.config"), haproxyConf.Bytes()); err != nil {
		return err
	}

	if err := createFile(path.Join(p.OutputDir, "conf", "error-page-404.http"), bytes.NewBuffer([]byte(error404)).Bytes()); err != nil {
		return err
	}

	return createFile(path.Join(p.OutputDir, "conf", "error-page-503.http"), bytes.NewBuffer([]byte(error503)).Bytes())
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
		if err := createFile(path.Join(p.OutputDir, "conf", m.MapName), m.Buffer.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func (c *GenProxyConfigCmd) generateCertConfig(p *ProgramCtx, backends []HAProxyBackendConfig) error {
	var certConfigMap bytes.Buffer

	for _, b := range filterBackendsByType([]TrafficType{EdgeTraffic, ReencryptTraffic}, backends) {
		if _, err := io.WriteString(&certConfigMap, fmt.Sprintf("%s %s\n", p.Certificate, b.Name)); err != nil {
			return err
		}
	}

	return createFile(path.Join(p.OutputDir, "conf", "cert_config.map"), certConfigMap.Bytes())
}

func (c *GenProxyConfigCmd) generateMBRequests(p *ProgramCtx, backends []HAProxyBackendConfig) error {
	for _, clients := range []int64{100} {
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
			for _, keepAliveRequests := range []int64{0, 50} {
				config := RequestConfig{
					Clients:           clients,
					KeepAliveRequests: keepAliveRequests,
					TLSSessionReuse:   c.TLSReuse,
					TrafficTypes:      scenario.TrafficTypes,
				}
				requests := generateMBRequests(config, filterBackendsByType(scenario.TrafficTypes, backends))
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
