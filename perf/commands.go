package main

import (
	"context"
	"time"
)

type Globals struct {
	Debug            bool        `help:"Enable debug mode" short:"D" default:"false"`
	DiscoveryURL     string      `help:"Backend metadata discovery URL" short:"u" default:"http://localhost:2000"`
	HTTPPort         int         `help:"HAProxy HTTP port" default:"8080"`
	HTTPSPort        int         `help:"HAProxy HTTPS port" default:"8443"`
	HTTPSPortSNIOnly int         `help:"HAProxy HTTPS port for SNI-only traffic" default:"9443"`
	HostPrefix       string      `help:"Hostname prefix" default:"perf-test-hydra"`
	Nbackends        int         `help:"Number of backends per traffic type" short:"n" default:"1"`
	OutputDir        string      `help:"Configuration output directory" short:"o" default:"testrun"`
	Port             int         `help:"Port number for backend metadata server" short:"p" default:"2000"`
	Profile          bool        `help:"Record a CPU profile." short:"P"`
	TLSReuse         bool        `help:"Enable TLS session reuse" default:"true"`
	Version          VersionFlag `help:"Print version information and quit."`
}

type CLI struct {
	Globals

	GenHosts       GenHostsCmd       `cmd:"" help:"Generate host names (/etc/hosts compatible)."`
	GenProxyConfig GenProxyConfigCmd `cmd:"" help:"Generate HAProxy configuration."`
	GenWorkload    GenWorkloadCmd    `cmd:"" help:"Generate https://github.com/jmencak/mb requests."`
	ServeBackend   ServeBackendCmd   `cmd:"" help:"Serve backend." hidden:"true"`
	ServeBackends  ServeBackendsCmd  `cmd:"" help:"Serve backends."`
	Test           TestCmd           `cmd:"" help:"Run client test using requests file."`
	Version        VersionCmd        `cmd:"" help:"Print version information and quit."`
}

type ProgramCtx struct {
	Context context.Context
	Globals
}

type TestCmd struct {
	Duration    time.Duration `help:"Test duration" short:"d" default:"60s"`
	RequestFile string        `help:"Request file." short:"i" type:"existingfile"`
}

type GenProxyConfigCmd struct {
	EnableLogging        bool   `default:"true"`
	ListenAddress        string `default:"::"`
	Maxconn              int    `default:"0"`
	Nthreads             int    `default:"4"`
	StatsPort            int    `default:"1936"`
	UseUnixDomainSockets bool   `default:"true"`
}

type GenHostsCmd struct {
	IPAddress string
}

type GenWorkloadCmd struct {
	UseProxy bool `default:"true"`
}

type ServeBackendsCmd struct {
	ListenAddress string `default:"127.0.0.1"`
}

type ServeBackendCmd struct {
	Name          string      `default:""`
	ListenAddress string      `default:""`
	TrafficType   TrafficType `default:""`
}

type VersionCmd struct{}
