package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/alecthomas/kong"
)

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

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if v, ok := os.LookupEnv("DISCOVERY_URL"); ok && v != "" {
		cli.Globals.DiscoveryURL = v
	}

	// This is to make paths emitted in the haproxy.config
	// absolute which helps with certificates.
	absPath, err := filepath.Abs(cli.Globals.OutputDir)
	if err != nil {
		log.Fatal(err)
	}
	cli.Globals.OutputDir = absPath

	if err := ktx.Run(&ProgramCtx{Globals: cli.Globals, Context: signalCtx}); err != nil {
		log.Fatal(err)
	}

	if cli.Globals.Debug {
		log.Println("number of goroutines:", runtime.NumGoroutine())
	}
}
