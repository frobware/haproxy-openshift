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
	Backends  int         `help:"Number of backends per traffic type" short:"b" default:"1"`
	Debug     bool        `help:"Enable debug mode" short:"D" default:"false"`
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
