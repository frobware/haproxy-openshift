package main

type CertStore struct {
	DomainFile    string
	RootCAFile    string
	RootCAKeyFile string
	TLSCertFile   string
	TLSKeyFile    string
}

type Backend struct {
	Name        string      `json:"name"`
	TrafficType TrafficType `json:"traffic_type"`
}

type BoundBackend struct {
	Backend

	ListenAddress string `json:"listen_address"`
	Port          int    `json:"port"`
}

type BackendsByTrafficType map[TrafficType][]Backend
type BoundBackendsByTrafficType map[TrafficType][]BoundBackend

// Multiple-host HTTP(s) Benchmarking tool
type MBRequest struct {
	Clients           int64  `json:"clients"`
	Host              string `json:"host"`
	KeepAliveRequests int64  `json:"keep-alive-requests"`
	Method            string `json:"method"`
	Path              string `json:"path"`
	Port              int64  `json:"port"`
	Scheme            string `json:"scheme"`
	TLSSessionReuse   bool   `json:"tls-session-reuse"`
}

type RequestConfig struct {
	Clients           int64
	KeepAliveRequests int64
	TLSSessionReuse   bool
	TrafficTypes      []TrafficType
}
