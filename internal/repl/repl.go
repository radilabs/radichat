package repl

import (
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
	ExitOK         = 0
	ExitError      = 1
	ExitInterrupt  = 130
	ExitTerminated = 143
	minInputBound  = 4096
	prompt         = "> "
)

// Options selects optional JSONL recording. A nil Records writer keeps the
// existing terminal-only behavior.
type Options struct {
	Client      *client.Client
	Records     RecordWriter
	JSONLStdout bool
}

func Run(ctx context.Context, cfg config.Config, in io.Reader, out, errw io.Writer) int {
	return RunWithOptions(ctx, cfg, in, out, errw, Options{})
}

func RunWithOptions(ctx context.Context, cfg config.Config, in io.Reader, out, errw io.Writer, opt Options) int {
	return runOpts(ctx, cfg, in, out, errw, opt)
}

func run(ctx context.Context, cfg config.Config, in io.Reader, out, errw io.Writer, cl *client.Client) int {
	return runOpts(ctx, cfg, in, out, errw, Options{Client: cl})
}

func runOpts(ctx context.Context, cfg config.Config, in io.Reader, out, errw io.Writer, opt Options) int {
	cl := opt.Client
	if cl == nil {
		cl = client.New(cfg, client.Options{})
	}
	conv := chat.New(cfg.ContextBudget, cfg.GenerationReserve, cfg.SystemPrompt)
	interactive := isTTY(in) && isTTY(out)
	maxLine := cfg.ContextBudget
	if maxLine < minInputBound {
		maxLine = minInputBound
	}
	reader := newInputReader(in, maxLine)
	failedTurn := false

	for {
		if err := ctx.Err(); err != nil {
			if conv.HasStaged() {
				conv.Abort()
			}
			return interruptExit(ctx)
		}
		if interactive {
			fmt.Fprint(errw, prompt)
		}
		got, err := reader.ReadLine(ctx)
		line := got.Text
		if errors.Is(err, errInterrupted) || errors.Is(err, context.Canceled) {
			if conv.HasStaged() {
				conv.Abort()
			}
			fmt.Fprintln(errw, "interrupted")
			return interruptExit(ctx)
		}
		if errors.Is(err, errLineTooLong) {
			fmt.Fprintf(errw, "input line exceeds %d bytes; shorten the message or raise context_budget\n", maxLine)
			failedTurn = true
			if err := opt.emit(Record{V: 1, Line: got.Number, Kind: "turn", Status: "error", Sent: boolPtr(false), Error: "line too long"}); err != nil {
				return ExitError
			}
			continue
		}
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Fprintf(errw, "cannot read input: %v\n", err)
			return ExitError
		}
		eof := errors.Is(err, io.EOF)
		if strings.TrimSpace(line) == "" {
			if eof {
				if failedTurn {
					return ExitError
				}
				return ExitOK
			}
			continue
		}
		if code, handled := handleCommand(line, conv, errw, opt, got.Number); handled {
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
			if err := opt.emit(Record{V: 1, Line: got.Number, Kind: "command_error", Input: strPtr(line), Error: "unknown command"}); err != nil {
				return ExitError
			}
			if eof {
				return ExitError
			}
			continue
		}

		msgs, trimmed, err := conv.Stage(line)
		if err != nil {
			fmt.Fprintf(errw, "%s\n", err.Error())
			failedTurn = true
			if werr := opt.emit(Record{V: 1, Line: got.Number, Kind: "turn", Status: "error", Sent: boolPtr(false), Input: strPtr(line), Error: err.Error()}); werr != nil {
				return ExitError
			}
			if eof {
				return ExitError
			}
			continue
		}
		if trimmed > 0 {
			fmt.Fprintf(errw, "dropped %d older turn(s) to stay within context_budget\n", trimmed)
		}

		var onDelta func(string) error
		if !opt.JSONLStdout {
			onDelta = func(delta string) error {
				_, werr := io.WriteString(out, delta)
				return werr
			}
		}
		res, streamErr := cl.StreamResult(ctx, msgs, conv.StreamLimit(), onDelta)
		if !opt.JSONLStdout {
			if len(res.Text) > 0 && !strings.HasSuffix(res.Text, "\n") {
				fmt.Fprintln(out)
			} else if len(res.Text) == 0 && streamErr == nil {
				fmt.Fprintln(out)
			}
		}
		if streamErr != nil {
			conv.Abort()
			if errors.Is(streamErr, context.Canceled) || errors.Is(streamErr, context.DeadlineExceeded) || ctx.Err() != nil {
				fmt.Fprintln(errw, "interrupted; in-flight turn discarded")
				_ = opt.emit(Record{V: 1, Line: got.Number, Kind: "turn", Status: "interrupted", Sent: boolPtr(res.RequestSent), Input: strPtr(line)})
				return interruptExit(ctx)
			}
			fmt.Fprintf(errw, "%s\n", streamErr.Error())
			failedTurn = true
			if werr := opt.emit(Record{V: 1, Line: got.Number, Kind: "turn", Status: "error", Sent: boolPtr(res.RequestSent), Input: strPtr(line), Error: streamErr.Error()}); werr != nil {
				return ExitError
			}
			if eof {
				return ExitError
			}
			continue
		}
		if err := conv.Commit(res.Text); err != nil {
			fmt.Fprintf(errw, "%s\n", err.Error())
			return ExitError
		}
		if werr := opt.emit(Record{
			V: 1, Line: got.Number, Kind: "turn", Status: "ok",
			Input: strPtr(line), Output: strPtr(res.Text),
			FinishReason: strPtr(res.FinishReason), TrimmedPairs: intPtr(trimmed),
		}); werr != nil {
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

func handleCommand(line string, conv *chat.Conversation, errw io.Writer, opt Options, lineNo int) (exitCode int, handled bool) {
	cmd := strings.TrimSpace(line)
	switch cmd {
	case "/quit":
		if err := opt.emit(Record{V: 1, Line: lineNo, Kind: "quit"}); err != nil {
			return ExitError, true
		}
		return ExitOK, true
	case "/clear":
		conv.Clear()
		fmt.Fprintln(errw, "conversation cleared")
		if err := opt.emit(Record{V: 1, Line: lineNo, Kind: "clear"}); err != nil {
			return ExitError, true
		}
		return -1, true
	default:
		return 0, false
	}
}

func (opt Options) emit(rec Record) error {
	if opt.Records == nil {
		return nil
	}
	return opt.Records.WriteRecord(rec)
}

var (
	errLineTooLong = errors.New("line too long")
	errInterrupted = errors.New("interrupted")
)

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
