package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/cli"
	"golang.org/x/term"
)

const draftDiagnosticGrace = 250 * time.Millisecond

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
	// Let stdout writes report EPIPE so the CLI can report the saved path.
	signal.Ignore(syscall.SIGPIPE)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	draft := len(args) >= 2 && args[0] == "health" && args[1] == "draft"
	var cancelDiagnostics context.CancelFunc
	if draft && app.Out != nil {
		app.Out = interruptibleOutput{ctx: ctx, writer: app.Out}
	}
	if draft && app.Err != nil {
		var diagnostics context.Context
		diagnostics, cancelDiagnostics = context.WithCancel(context.Background())
		defer cancelDiagnostics()
		app.Err = interruptibleOutput{ctx: diagnostics, writer: app.Err}
	}
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	defer signal.Stop(interrupts)
	// Restore normal signal behavior after the first interrupt, so a second one
	// can terminate blocked finalization.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-interrupts:
			signal.Stop(interrupts)
			cancel()
			if cancelDiagnostics != nil {
				// Give writable stderr time to report retained files and pending
				// selections, with one deadline for all diagnostic writes.
				timer := time.NewTimer(draftDiagnosticGrace)
				defer timer.Stop()
				select {
				case <-timer.C:
					cancelDiagnostics()
				case <-done:
				}
			}
		case <-done:
		}
	}()
	code := app.Run(ctx, args)
	if draft && ctx.Err() != nil {
		return 130
	}
	return code
}

type interruptibleOutput struct {
	ctx    context.Context
	writer io.Writer
}

func (o interruptibleOutput) Write(p []byte) (int, error) {
	if err := o.ctx.Err(); err != nil {
		return 0, err
	}
	type result struct {
		n   int
		err error
	}
	// An inherited stream can block in the OS. The process can still exit,
	// which ends the outstanding write.
	// Copy the bytes because the caller may reuse its buffer after cancellation.
	data := bytes.Clone(p)
	done := make(chan result, 1)
	go func() {
		n, err := o.writer.Write(data)
		done <- result{n, err}
	}()
	select {
	case r := <-done:
		return r.n, r.err
	case <-o.ctx.Done():
		return 0, o.ctx.Err()
	}
}
