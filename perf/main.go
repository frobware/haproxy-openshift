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

	// This is to make paths in haproxy.config absolute.
	absPath, err := filepath.Abs(cli.Globals.OutputDir)
	if err != nil {
		log.Fatal(err)
	}
	cli.Globals.OutputDir = absPath

	ktx.FatalIfErrorf(ktx.Run(&ProgramCtx{Globals: cli.Globals, Context: ctx}))
}
