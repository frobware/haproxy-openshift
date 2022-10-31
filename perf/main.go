package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/alecthomas/kong"
)

type Globals struct {
	Debug        bool        `help:"Enable debug mode" short:"D" default:"false"`
	DiscoveryURL string      `help:"Backend metadata discovery URL" short:"u" default:"http://localhost:2000"`
	HostPrefix   string      `help:"Hostname prefix" default:"perf-test-hydra"`
	Nbackends    int         `help:"Number of backends per traffic type" short:"n" default:"1"`
	OutputDir    string      `help:"Configuration output directory" short:"o" default:"testrun"`
	Port         int         `help:"Port number for backend metadata server" short:"p" default:"2000"`
	Version      VersionFlag `help:"Print version information and quit."`
}

type CLI struct {
	Globals

	GenProxyConfig GenProxyConfigCmd `cmd:"" help:"Generate HAProxy configuration."`
	GenHosts       GenHostsCmd       `cmd:"" help:"Generate host names (/etc/hosts compatible)."`
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

	if v, ok := os.LookupEnv("DISCOVERY_URL"); ok && v != "" {
		cli.Globals.DiscoveryURL = v
	}

	// This is to fix cert locations in the haproxy.config.
	absPath, err := filepath.Abs(cli.Globals.OutputDir)
	if err != nil {
		log.Fatal(err)
	}
	cli.Globals.OutputDir = absPath

	ktx.FatalIfErrorf(ktx.Run(&ProgramCtx{Globals: cli.Globals, Context: ctx}))
}
