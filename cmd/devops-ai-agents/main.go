package main

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/patryk-plonka/devops-ai-agents/internal/cli"
	"golang.org/x/term"
)

func main() { os.Exit(run()) }
func run() int {
	dir, err := os.Getwd()
	if err != nil {
		_, _ = os.Stderr.WriteString("Cannot determine working directory.\n")
		return 1
	}
	version := "development"
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		version = info.Main.Version
	}
	return runApp(cli.App{In: os.Stdin, Out: os.Stdout, Err: os.Stderr, Terminal: term.IsTerminal(int(os.Stdin.Fd())), Directory: dir, Version: version}, os.Args[1:])
}

func runApp(app cli.App, args []string) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)
	// Restore normal signal behavior after the first interrupt, so a second one
	// can terminate blocked finalization. Local Save/Publish do not use ctx.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-interrupts:
			signal.Stop(interrupts)
			cancel()
		case <-done:
		}
	}()
	return app.Run(ctx, args)
}
