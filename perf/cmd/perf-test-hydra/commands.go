package main

import (
	"fmt"
	"os"

	"github.com/frobware/haproxy-openshift/perf/pkg/termination"
)

type Globals struct {
	Backends  int         `help:"Number of backends" short:"b" default:"1"`
	Debug     bool        `help:"Enable debug mode" short:"D"`
	OutputDir string      `help:"Output directory" short:"o" default:"/tmp/perf-test-hydra"`
	Port      int         `help:"Port number for backend metadata server" short:"p" default:"2000"`
	TLSCACert string      `help:"Trust certs signed only by this CA" default:"ca.pem" type:"path"`
	TLSCert   string      `help:"Path to TLS certificate file" default:"cert.pem" type:"path"`
	TLSKey    string      `help:"Path to TLS key file" default:"key.pem" type:"path"`
	Version   VersionFlag `help:"Print version information and quit."`
}

type CLI struct {
	Globals

	ServeBackends ServeBackendsCmd `cmd:"" help:"Serve backends."`
	ServeBackend  ServeBackendCmd  `cmd:"" help:"Serve child backend." hidden:"true"`
	Version       VersionCmd       `cmd:"" help:"Print the version information."`
}

type ServeBackendCmd struct {
	Args []string `arg:""`
}

type ServeBackendsCmd struct{}

func (c *ServeBackendCmd) Run(p *ProgramCtx) error {
	backend, found := os.LookupEnv(ChildBackendEnvName)
	if !found {
		return fmt.Errorf("%q not found in environment", ChildBackendEnvName)
	}

	trafficType, found := os.LookupEnv(ChildBackendTrafficTypeEnvName)
	if !found {
		return fmt.Errorf("%q not found in environment", ChildBackendTrafficTypeEnvName)
	}

	// never returns
	return serveBackend(backend, trafficType, p.Port)
}

func (c *ServeBackendsCmd) Run(p *ProgramCtx) error {
	hostIPAddr := mustResolveCurrentHost()
	allBackends := BackendsByTrafficType{}

	for _, t := range termination.AllTerminationTypes {
		for i := 0; i < p.Backends; i++ {
			backend := Backend{
				HostAddr:    hostIPAddr,
				Name:        fmt.Sprintf("%s-%v-%v", "ocp-http-scale", t, i),
				TrafficType: t,
			}
			allBackends[t] = append(allBackends[t], backend)
		}
	}

	startBackends(p.Context, allBackends, p.Port)
	return nil
}
