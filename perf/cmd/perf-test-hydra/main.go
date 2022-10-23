package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
)

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
		log.Fatal("received signal, cancelling", s)
	}()

	// ktx.BindTo(ctx, (*context.Context)(nil))

	ktx.FatalIfErrorf(ktx.Run(&ProgramCtx{Globals: cli.Globals, Context: ctx}))
}
