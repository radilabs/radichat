// Command radichat-mock runs a deterministic, loopback-only mock of an
// OpenAI-compatible Chat Completions endpoint for testing RadiChat.
//
// It is a development tool. It is not part of the RadiChat product, is not
// included in release binaries or container images, and binds only to loopback.
//
// Usage:
//
//	radichat-mock list                      list named scenarios
//	radichat-mock run <scenario>            serve one scenario until interrupted
//	radichat-mock run <scenario> -addr HOST:PORT
//	                                      serve on an explicit loopback address
//
// When a scenario starts, the CLI prints the endpoint URL to configure in
// radichat.json. It exits cleanly on SIGINT or SIGTERM.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/radilabs/radichat/internal/mockllm"
)

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		usage(stdout)
		return 2
	}
	switch args[1] {
	case "list":
		return listScenarios(stdout)
	case "run":
		return runScenario(args[2:], stdout, stderr)
	case "-h", "-help", "--help", "help":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "radichat-mock: unknown command %q; see radichat-mock -help\n", args[1])
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `radichat-mock — loopback-only mock of an OpenAI-compatible chat endpoint

Usage:
  radichat-mock list                 list named scenarios
  radichat-mock run <scenario>       serve one scenario until interrupted
  radichat-mock run <scenario> -addr HOST:PORT
                                   serve on an explicit loopback address
  radichat-mock -help               show this help
The mock binds loopback only. Point radichat.json at the printed endpoint URL.
See docs/testing-mock.md for reproducible commands and examples.
`)
}

// listScenarios prints the scenario catalog grouped by family.
func listScenarios(w io.Writer) int {
	catalog := mockllm.Catalog()
	if len(catalog) == 0 {
		fmt.Fprintln(w, "no scenarios registered")
		return 1
	}
	fmt.Fprintf(w, "%d scenarios:\n\n", len(catalog))
	for _, family := range mockllm.Families() {
		fmt.Fprintf(w, "%s\n", family)
		for _, s := range catalog {
			if s.Family != family {
				continue
			}
			fmt.Fprintf(w, "  %-40s %s\n", s.Name, s.Description)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "Run one with: radichat-mock run <scenario>\n")
	return 0
}

// runScenario serves one named scenario until interrupted.
//
// Flags are accepted both before and after the scenario name. Go's flag package
// stops parsing at the first non-flag argument, so the arguments are reordered
// first: everything that looks like a flag is moved ahead of the positional
// scenario name.
func runScenario(args []string, stdout, stderr io.Writer) int {
	flags, positional := splitFlags(args)
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", "", "loopback listen address, for example 127.0.0.1:0 (default: an ephemeral loopback port)")
	if err := fs.Parse(flags); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	rest := append(fs.Args(), positional...)
	if len(rest) == 0 {
		fmt.Fprintln(stderr, "radichat-mock: run needs a scenario name; list them with radichat-mock list")
		return 2
	}
	name := rest[0]
	sc, ok := mockllm.Lookup(name)
	if !ok {
		fmt.Fprintf(stderr, "radichat-mock: unknown scenario %q; list scenarios with radichat-mock list\n", name)
		return 2
	}

	listenAddr := *addr
	if listenAddr == "" {
		listenAddr = mockllm.DefaultAddr
	}
	if err := checkLoopbackAddr(listenAddr); err != nil {
		fmt.Fprintf(stderr, "radichat-mock: %s\n", err.Error())
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv, err := mockllm.StartContext(ctx, mockllm.Options{
		Addr:           listenAddr,
		ExpectedBearer: mockllm.SentinelToken,
	})
	if err != nil {
		fmt.Fprintf(stderr, "radichat-mock: %s\n", err.Error())
		return 1
	}

	url := srv.URL() + sc.Path()
	fmt.Fprintf(stdout, "scenario:   %s\n", sc.Name)
	fmt.Fprintf(stdout, "family:     %s\n", sc.Family)
	fmt.Fprintf(stdout, "behavior:   %s\n", sc.Description)
	if sc.Auth.Required {
		fmt.Fprintf(stdout, "auth:       requires a bearer token (sentinel: %s)\n", mockllm.SentinelEnvName)
	} else {
		fmt.Fprintf(stdout, "auth:       none required\n")
	}
	fmt.Fprintf(stdout, "endpoint:   %s\n", url)
	fmt.Fprintln(stdout, "listening on loopback; press Ctrl-C to stop")

	<-ctx.Done()
	fmt.Fprintln(stdout, "shutting down")
	if err := srv.Close(); err != nil {
		fmt.Fprintf(stderr, "radichat-mock: shutdown: %s\n", err.Error())
		return 1
	}
	return 0
}

// splitFlags separates flag-looking arguments from positional ones so the flag
// parser sees them regardless of order. A "--" terminator ends flag parsing and
// keeps the remainder positional.
func splitFlags(args []string) (flags, positional []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			// Everything after the terminator is positional, including
			// flag-looking values.
			return flags, append(positional, args[i+1:]...)
		case strings.HasPrefix(a, "-") && a != "-":
			flags = append(flags, a)
			// A flag written as "-addr value" consumes the next argument.
			if !strings.Contains(a, "=") && i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}
		default:
			positional = append(positional, a)
		}
	}
	return flags, positional
}

// checkLoopbackAddr rejects a non-loopback -addr value before binding.
func checkLoopbackAddr(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid -addr %q: use HOST:PORT", addr)
	}
	if port != "" {
		if _, err := strconv.Atoi(port); err != nil {
			return fmt.Errorf("invalid -addr port %q: use a numeric port", port)
		}
	}
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "localhost", "127.0.0.1", "::1", "0:0:0:0:0:0:0:1":
		return nil
	}
	if ip := net.ParseIP(strings.TrimSpace(host)); ip != nil && ip.IsLoopback() {
		return nil
	}
	return errors.New("refusing to bind a non-loopback address; use 127.0.0.1, [::1], or localhost")
}
