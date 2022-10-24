package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strings"
)

type HAProxyGenCmd struct {
	HTTPPort  int  `default:"8080"`
	HTTPSPort int  `default:"8443"`
	Maxconn   int  `default:"0"`
	Nthreads  int  `default:"4"`
	StatsPort int  `default:"1936"`
	TLSReuse  bool `default:"true"`
}

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

type HAProxyConfig struct {
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
	TrafficType   TrafficType
}

const (
	HTTPBackendMapName      = "os_http_be.map"
	ReencryptBackendMapName = "os_edge_reencrypt_be.map"
	SNIPassthroughMapName   = "os_sni_passthrough.map"
	TCPBackendMapName       = "os_tcp_be.map"
)

//go:embed globals.tmpl
var globalTemplate string

//go:embed defaults.tmpl
var defaultTemplate string

//go:embed backends.tmpl
var backendTemplate string

//go:embed error-page-404.http
var error404 string

//go:embed error-page-503.http
var error503 string

var (
	discoveryURL = flag.String("discovery", "http://localhost:2000", "backend discovery URL")
	httpPort     = flag.Int("http-port", 8080, "haproxy http port setting")
	httpsPort    = flag.Int("https-port", 8443, "haproxy https port setting")
	maxconn      = flag.Int("maxconn", 0, "haproxy maxconn setting")
	nbthread     = flag.Int("nbthread", 4, "haproxy nbthread setting")
	statsPort    = flag.Int("stats-port", 1936, "haproxy https port setting")
	tlsreuse     = flag.Bool("tlsreuse", true, "enable TLS reuse")
)

func cookie() string {
	runes := []rune("0123456789abcdef")
	b := make([]rune, 32)
	for i := 0; i < 32; i++ {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}

func fetchBackendMetadata[T TrafficType](t T) ([]string, error) {
	url := fmt.Sprintf("%s/backends/%v", *discoveryURL, t)
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

func generateHAProxyBackendConfig(path string) ([]HAProxyBackendConfig, error) {
	var backends []HAProxyBackendConfig

	for _, t := range AllTrafficTypes {
		metadata, err := fetchBackendMetadata(t)
		if err != nil {
			return nil, err
		}
		for i := range metadata {
			words := strings.Split(metadata[i], " ")
			if len(words) < 3 {
				return nil, fmt.Errorf("not enough words in %q", metadata[i])
			}
			backends = append(backends, HAProxyBackendConfig{
				BackendCookie: cookie(),
				HostAddr:      words[0],
				Name:          words[1],
				OutputDir:     path,
				Port:          words[2],
				ServerCookie:  cookie(),
				TrafficType:   t,
			})
		}
	}

	return backends, nil
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

func generateMBRequests(request RequestConfig, backends []HAProxyBackendConfig) []Request {
	var requests []Request

	for _, b := range backends {
		requests = append(requests, Request{
			Clients:           request.Clients,
			Host:              fmt.Sprintf("%s", b.Name),
			KeepAliveRequests: request.KeepAliveRequests,
			Method:            "GET",
			Path:              "/1024.html",
			Port:              b.TrafficType.Port(),
			Scheme:            b.TrafficType.Scheme(),
			TLSSessionReuse:   *tlsreuse,
		})
	}

	return requests
}

func (c *HAProxyGenCmd) Run(p *ProgramCtx) error {
	if err := os.RemoveAll(p.OutputDir); err != nil {
		return err
	}

	allBackends, err := generateHAProxyBackendConfig(p.OutputDir)
	if err != nil {
		return err
	}

	config := HAProxyConfig{
		Backends:  allBackends,
		HTTPPort:  *httpPort,
		HTTPSPort: *httpsPort,
		Maxconn:   *maxconn,
		Nbthread:  *nbthread,
		OutputDir: p.OutputDir,
		StatsPort: *statsPort,
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

	type MapEntryFunc func(backend HAProxyBackendConfig) string

	maps := []struct {
		Filename     string
		TrafficTypes []TrafficType
		MapEntry     MapEntryFunc
		Buffer       *bytes.Buffer
	}{{
		Filename:     HTTPBackendMapName,
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
		Filename:     ReencryptBackendMapName,
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
		Filename:     SNIPassthroughMapName,
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
		Filename:     TCPBackendMapName,
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
	}}

	if err := createFile(path.Join(p.OutputDir, "conf", "haproxy.config"), haproxyConf.Bytes()); err != nil {
		return err
	}

	if err := createFile(path.Join(p.OutputDir, "conf", "error-page-404.http"), bytes.NewBuffer([]byte(error404)).Bytes()); err != nil {
		return err
	}

	if err := createFile(path.Join(p.OutputDir, "conf", "error-page-503.http"), bytes.NewBuffer([]byte(error503)).Bytes()); err != nil {
		return err
	}

	for _, m := range maps {
		for _, b := range filterBackendsByType(m.TrafficTypes, allBackends) {
			if _, err := io.WriteString(m.Buffer, m.MapEntry(b)); err != nil {
				return err
			}
		}
		if err := createFile(path.Join(p.OutputDir, "conf", m.Filename), m.Buffer.Bytes()); err != nil {
			return err
		}
	}

	for _, clients := range []int64{1, 50, 100, 200} {
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
			for _, keepAliveRequests := range []int64{0, 1, 50} {
				config := RequestConfig{
					Clients:           clients,
					KeepAliveRequests: keepAliveRequests,
					TLSSessionReuse:   false,
					TrafficTypes:      scenario.TrafficTypes,
				}
				requests := generateMBRequests(config, filterBackendsByType(scenario.TrafficTypes, allBackends))
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
					log.Fatalf("error: failed to create path: %q: %v", path, err)
				}
				filename := fmt.Sprintf("%s/requests.json", path)
				fmt.Println(filename)
				if err := createFile(filename, data); err != nil {
					log.Fatalf("error generating %s: %v", filename, err)
				}
			}
		}
	}

	return nil
}