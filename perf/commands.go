package main

type GenProxyConfigCmd struct {
	HTTPPort             int    `default:"8080"`
	HTTPSPort            int    `default:"8443"`
	ListenAddress        string `default:"::"`
	Maxconn              int    `default:"0"`
	Nthreads             int    `default:"4"`
	StatsPort            int    `default:"1936"`
	TLSReuse             bool   `default:"true"`
	UseUnixDomainSockets bool   `default:"true"`
}

type GenHostsCmd struct {
	IPAddress string
}

type GenWorkloadCmd struct {
	TLSReuse bool `default:"true"`
}

type ServeBackendsCmd struct {
	ListenAddress string `default:"127.0.0.1"`
}

type ServeBackendCmd struct {
	Name          string      `default:"<error>"`
	ListenAddress string      `default:"<error>"`
	TrafficType   TrafficType `default:"<error>"`
}

type VersionCmd struct{}
