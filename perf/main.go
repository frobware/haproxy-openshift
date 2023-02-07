package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
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

	if cli.Globals.Profile {
		pprofFile, pprofErr := os.Create("cpu.pprof")
		if pprofErr != nil {
			log.Fatal(pprofErr)
		}
		pprof.StartCPUProfile(pprofFile)
		defer func() {
			log.Println("stopping CPU profile")
			pprof.StopCPUProfile()
		}()
	}

	if err := ktx.Run(&ProgramCtx{Globals: cli.Globals, Context: signalCtx}); err != nil {
		log.Fatal(err)
	}

	if cli.Globals.Debug {
		log.Println("number of goroutines:", runtime.NumGoroutine())
	}
}
