package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bifshteks/mur/internal/cli"
)

// Set at build time with -ldflags "-X main.version=... -X main.buildTime=...".
//
//nolint:gochecknoglobals // ldflags targets
var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	// Interrupting cancels the context, which stops any git clone or fetch in
	// flight before mur exits.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	dir, _ := os.Getwd()
	code := cli.Run(ctx, cli.Env{
		Args:      os.Args[1:],
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
		Stderr:    os.Stderr,
		IsTTY:     isTerminal(os.Stdin),
		Environ:   os.Environ(),
		Dir:       dir,
		Version:   version,
		BuildTime: buildTime,
	})

	stop()
	os.Exit(code)
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()

	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}
