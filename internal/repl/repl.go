package repl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/radilabs/radichat/internal/chat"
	"github.com/radilabs/radichat/internal/client"
	"github.com/radilabs/radichat/internal/config"
)

const (
	ExitOK        = 0
	ExitError     = 1
	ExitInterrupt = 130
	minInputBound = 4096
	prompt        = "> "
)

func Run(ctx context.Context, cfg config.Config, in io.Reader, out, errw io.Writer) int {
	return run(ctx, cfg, in, out, errw, nil)
}

func run(ctx context.Context, cfg config.Config, in io.Reader, out, errw io.Writer, cl *client.Client) int {
	if cl == nil {
		cl = client.New(cfg, client.Options{})
	}
	conv := chat.New(cfg.ContextBudget, cfg.GenerationReserve, cfg.SystemPrompt)
	interactive := isTTY(in) && isTTY(out)
	br := bufio.NewReader(in)
	maxLine := cfg.ContextBudget
	if maxLine < minInputBound {
		maxLine = minInputBound
	}
	failedTurn := false

	for {
		if err := ctx.Err(); err != nil {
			if conv.HasStaged() {
				conv.Abort()
			}
			return ExitInterrupt
		}
		if interactive {
			fmt.Fprint(errw, prompt)
		}
		line, err := readLine(ctx, br, maxLine)
		if errors.Is(err, errInterrupted) || errors.Is(err, context.Canceled) {
			if conv.HasStaged() {
				conv.Abort()
			}
			fmt.Fprintln(errw, "interrupted")
			return ExitInterrupt
		}
		if errors.Is(err, errLineTooLong) {
			fmt.Fprintf(errw, "input line exceeds %d bytes; shorten the message or raise context_budget\n", maxLine)
			failedTurn = true
			continue
		}
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Fprintf(errw, "cannot read input: %v\n", err)
			return ExitError
		}
		eof := errors.Is(err, io.EOF)
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			if eof {
				if failedTurn {
					return ExitError
				}
				return ExitOK
			}
			continue
		}
		if code, handled := handleCommand(line, conv, errw); handled {
			if code >= 0 {
				return code
			}
			if eof {
				if failedTurn {
					return ExitError
				}
				return ExitOK
			}
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "/") {
			fmt.Fprintf(errw, "unknown command %q; known commands are /clear and /quit\n", strings.TrimSpace(line))
			failedTurn = true
			if eof {
				return ExitError
			}
			continue
		}

		msgs, trimmed, err := conv.Stage(line)
		if err != nil {
			fmt.Fprintf(errw, "%s\n", err.Error())
			failedTurn = true
			if eof {
				return ExitError
			}
			continue
		}
		if trimmed > 0 {
			fmt.Fprintf(errw, "dropped %d older turn(s) to stay within context_budget\n", trimmed)
		}

		text, streamErr := cl.Stream(ctx, msgs, conv.StreamLimit(), func(delta string) error {
			if _, werr := io.WriteString(out, delta); werr != nil {
				return werr
			}
			return nil
		})
		if len(text) > 0 && !strings.HasSuffix(text, "\n") {
			fmt.Fprintln(out)
		} else if len(text) == 0 && streamErr == nil {
			fmt.Fprintln(out)
		}
		if streamErr != nil {
			conv.Abort()
			if errors.Is(streamErr, context.Canceled) || errors.Is(streamErr, context.DeadlineExceeded) || ctx.Err() != nil {
				fmt.Fprintln(errw, "interrupted; in-flight turn discarded")
				return ExitInterrupt
			}
			fmt.Fprintf(errw, "%s\n", streamErr.Error())
			failedTurn = true
			if eof {
				return ExitError
			}
			continue
		}
		if err := conv.Commit(text); err != nil {
			fmt.Fprintf(errw, "%s\n", err.Error())
			return ExitError
		}
		if eof {
			if failedTurn {
				return ExitError
			}
			return ExitOK
		}
	}
}

func handleCommand(line string, conv *chat.Conversation, errw io.Writer) (exitCode int, handled bool) {
	cmd := strings.TrimSpace(line)
	switch cmd {
	case "/quit":
		return ExitOK, true
	case "/clear":
		conv.Clear()
		fmt.Fprintln(errw, "conversation cleared")
		return -1, true
	default:
		return 0, false
	}
}

var (
	errLineTooLong = errors.New("line too long")
	errInterrupted = errors.New("interrupted")
)

func readLine(ctx context.Context, br *bufio.Reader, max int64) (string, error) {
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := readBoundedLine(br, max)
		ch <- result{line, err}
	}()
	select {
	case <-ctx.Done():
		return "", errInterrupted
	case r := <-ch:
		return r.line, r.err
	}
}

func readBoundedLine(br *bufio.Reader, max int64) (string, error) {
	var b []byte
	for {
		c, err := br.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(b) == 0 {
					return "", io.EOF
				}
				return string(b), io.EOF
			}
			return "", err
		}
		if c == '\n' {
			if len(b) > 0 && b[len(b)-1] == '\r' {
				b = b[:len(b)-1]
			}
			return string(b), nil
		}
		if int64(len(b)) >= max {
			for {
				c2, err2 := br.ReadByte()
				if err2 != nil || c2 == '\n' {
					break
				}
			}
			return "", errLineTooLong
		}
		b = append(b, c)
	}
}

func isTTY(w interface{}) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
