package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
)

type Globals struct {
	Debug        bool        `help:"Enable debug mode" short:"D" default:"false"`
	DiscoveryURL string      `help:"Backend metadata discovery URL" short:"u" default:"http://localhost:2000"`
	HostPrefix   string      `help:"Hostname prefix" default:"openshift-http-scale"`
	Nbackends    int         `help:"Number of backends per traffic type" short:"n" default:"1"`
	OutputDir    string      `help:"Output directory" short:"o" default:"/tmp/perf-test-hydra"`
	Port         int         `help:"Port number for backend metadata server" short:"p" default:"2000"`
	TLSCACert    string      `help:"Trust certs signed only by this CA" default:"rootCA.pem" type:"path"`
	TLSCert      string      `help:"Path to TLS certificate file" default:"tls.crt" type:"path"`
	TLSKey       string      `help:"Path to TLS key file" default:"tls.key" type:"path"`
	Certificate  string      `help:"Path to full chain certificate " default:"full-chain.pem" type:"path"`
	Version      VersionFlag `help:"Print version information and quit."`
}

type CLI struct {
	Globals

	HaproxyGen    HAProxyGenCmd    `cmd:"" help:"Generate HAProxy configuration."`
	PrintHosts    PrintHostsCmd    `cmd:"" help:"Print backend hostnames (/etc/hosts compatible)."`
	ServeBackend  ServeBackendCmd  `cmd:"" help:"Serve backend." hidden:"true"`
	ServeBackends ServeBackendsCmd `cmd:"" help:"Serve backends."`
	Version       VersionCmd       `cmd:"" help:"Print the version information."`
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

	ktx.FatalIfErrorf(ktx.Run(&ProgramCtx{Globals: cli.Globals, Context: ctx}))
}
