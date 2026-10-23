package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/radilabs/radichat/internal/chat"
	"github.com/radilabs/radichat/internal/config"
	"github.com/radilabs/radichat/internal/repl"
)

func main() {
	cause := repl.NewInterruptCause()
	ctx, stop := context.WithCancel(repl.WithInterruptCause(context.Background(), cause))
	defer stop()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)
	go func() {
		select {
		case sig := <-sigs:
			cause.Store(sig)
			stop()
		case <-ctx.Done():
		}
	}()
	os.Exit(run(ctx, os.Args, os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, in io.Reader, out, errw io.Writer) int {
	fs := flag.NewFlagSet("radichat", flag.ContinueOnError)
	fs.SetOutput(errw)
	configPath := fs.String("config", config.DefaultPath, "path to JSON configuration file")
	inputPath := fs.String("input", "", "read conversation lines from PATH instead of stdin")
	outputPath := fs.String("output", "", "write JSONL results to PATH, or - for stdout")
	fs.Usage = func() {
		fmt.Fprintf(out, "Usage: radichat [-config PATH] [-input PATH] [-output PATH|-]\n")
		fmt.Fprintf(out, "  RadiChat is a small terminal client for OpenAI-compatible chat endpoints.\n")
		fmt.Fprintf(out, "  Local use needs no token. Optional bearer auth names an environment variable in the config file.\n")
		fmt.Fprintf(out, "  Default config path: %s\n", config.DefaultPath)
		fmt.Fprintf(out, "  A file literally named - is passed as ./-. There is no -input - stdin alias.\n\n")
		fmt.Fprintf(out, "Flags:\n")
		fmt.Fprintf(out, "  -config PATH   JSON configuration file (default %s)\n", config.DefaultPath)
		fmt.Fprintf(out, "  -input PATH    read conversation lines from PATH instead of stdin\n")
		fmt.Fprintf(out, "  -output PATH   write JSONL results to PATH, or - for stdout\n")
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
	if err := systemPromptFits(cfg); err != nil {
		fmt.Fprintf(errw, "radichat: %s\n", err.Error())
		return 1
	}
	if *inputPath != "" {
		f, err := openInput(*inputPath)
		if err != nil {
			fmt.Fprintf(errw, "radichat: %s\n", err.Error())
			return 1
		}
		defer f.Close()
		in = f
	}
	// JSONL output is created only after config, fit, and input validation.
	var records repl.RecordWriter
	jsonlStdout := false
	if *outputPath != "" {
		if *outputPath == "-" {
			records = repl.NewJSONLWriter(out)
			jsonlStdout = true
		} else {
			f, err := createOutput(*outputPath)
			if err != nil {
				fmt.Fprintf(errw, "radichat: %s\n", err.Error())
				return 1
			}
			defer f.Close()
			records = repl.NewJSONLWriter(f)
		}
	}

	return repl.RunWithOptions(ctx, cfg, in, out, errw, repl.Options{
		Records:     records,
		JSONLStdout: jsonlStdout,
	})
}

func createOutput(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("output file %q already exists", path)
		}
		return nil, fmt.Errorf("cannot create output file %q: %w", path, err)
	}
	return f, nil
}

func openInput(path string) (*os.File, error) {
	if path == "-" {
		path = "./-"
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("input file %q not found", path)
		}
		return nil, fmt.Errorf("cannot read input file %q: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("input path %q is a directory", path)
	}
	if !info.Mode().IsRegular() && info.Mode()&os.ModeNamedPipe == 0 {
		return nil, fmt.Errorf("input path %q is not a regular file or FIFO", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read input file %q: %w", path, err)
	}
	return f, nil
}

// systemPromptFits applies the context-fit policy to a non-empty system prompt
// before any request. An empty or absent prompt keeps the previous behavior.
func systemPromptFits(cfg config.Config) error {
	if cfg.SystemPrompt == "" {
		return nil
	}
	ok, err := chat.Fits([]chat.Message{
		{Role: "system", Content: cfg.SystemPrompt},
		{Role: "user", Content: ""},
	}, cfg.ContextBudget, cfg.GenerationReserve)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("system prompt does not fit in context_budget with generation_reserve; shorten the prompt, reduce generation_reserve, or increase context_budget")
	}
	return nil
}
