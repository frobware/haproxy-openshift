package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
)

type Globals struct {
	Certificate  string      `help:"Path to certificate" default:"certs/domain.pem" type:"path"`
	Debug        bool        `help:"Enable debug mode" short:"D" default:"false"`
	DiscoveryURL string      `help:"Backend metadata discovery URL" short:"u" default:"http://localhost:2000"`
	HostPrefix   string      `help:"Hostname prefix" default:"perf-test-hydra"`
	MkCert       string      `help:"Path to mkcert script" default:"mkcert.bash" type:"path"`
	Nbackends    int         `help:"Number of backends per traffic type" short:"n" default:"1"`
	OutputDir    string      `help:"Output directory" short:"o" default:"/tmp/perf-test-hydra"`
	Port         int         `help:"Port number for backend metadata server" short:"p" default:"2000"`
	TLSCACert    string      `help:"Trust certs signed only by this CA" default:"certs/rootCA.pem" type:"path"`
	TLSCert      string      `help:"Path to TLS certificate file" default:"certs/tls.crt" type:"path"`
	TLSKey       string      `help:"Path to TLS key file" default:"certs/tls.key" type:"path"`
	Version      VersionFlag `help:"Print version information and quit."`
}

type CLI struct {
	Globals

	GenProxyConfig GenProxyConfigCmd `cmd:"" help:"Generate HAProxy configuration."`
	GenCerts       GenCertsCmd       `cmd:"" help:"Generate certificates."`
	GenHosts       GenHostsCmd       `cmd:"" help:"Generate host names (/etc/hosts compatible)."`
	GenWorkload    GenWorkloadCmd    `cmd:"" help:"Generate https://github.com/jmencak/mb requests."`
	ServeBackend   ServeBackendCmd   `cmd:"" help:"Serve backend." hidden:"true"`
	ServeBackends  ServeBackendsCmd  `cmd:"" help:"Serve backends."`
	Version        VersionCmd        `cmd:"" help:"Print version information and quit."`
}

type ProgramCtx struct {
	Context context.Context
	Globals
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)

	cli := CLI{
		Globals: Globals{
			Version: VersionFlag("0.1.1"),
		},
	}

	ktx := kong.Parse(&cli,
		kong.Name("perf-test-hydra"),
		kong.Description("Supposedly simplifies things."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
			Summary: true,
		}),
		kong.Vars{
			"version": "0.0.1",
		},
	)

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()
		s := <-sigc
		if cli.Globals.Debug {
			log.Printf("exiting on signal %v\n", s)
		}
		os.Exit(0)
	}()

	if _, err := os.Stat(cli.Globals.Certificate); errors.Is(err, os.ErrNotExist) {
		if err := generateCerts(&ProgramCtx{Globals: cli.Globals, Context: ctx}, true); err != nil {
			log.Fatal(err)
		}
	}

	ktx.FatalIfErrorf(ktx.Run(&ProgramCtx{Globals: cli.Globals, Context: ctx}))
}
