package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/radilabs/radichat/internal/config"
	"github.com/radilabs/radichat/internal/repl"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args, os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, in io.Reader, out, errw io.Writer) int {
	fs := flag.NewFlagSet("radichat", flag.ContinueOnError)
	fs.SetOutput(errw)
	configPath := fs.String("config", config.DefaultPath, "path to JSON configuration file")
	fs.Usage = func() {
		fmt.Fprintf(out, "Usage: radichat [-config PATH]\n")
		fmt.Fprintf(out, "  RadiChat is a small terminal client for unauthenticated OpenAI-compatible chat endpoints.\n")
		fmt.Fprintf(out, "  Default config path: %s\n\n", config.DefaultPath)
		fmt.Fprintf(out, "Flags:\n")
		fmt.Fprintf(out, "  -config PATH   JSON configuration file (default %s)\n", config.DefaultPath)
		fmt.Fprintf(out, "  -help          show this help and exit\n")
	}
	if err := fs.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(errw, "radichat: unexpected argument %q; see -help\n", fs.Arg(0))
		return 2
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(errw, "radichat: %s\n", err.Error())
		return 1
	}

	return repl.Run(ctx, cfg, in, out, errw)
}
