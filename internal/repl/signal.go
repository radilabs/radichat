package repl

import (
	"context"
	"os"
	"sync/atomic"
	"syscall"
)

// InterruptCause records which signal canceled the process. It is stored
// before the context is canceled so the REPL can choose exit 130 or 143.
type InterruptCause struct {
	sig atomic.Value
}

func NewInterruptCause() *InterruptCause {
	return &InterruptCause{}
}

func (c *InterruptCause) Store(sig os.Signal) {
	if c == nil || sig == nil {
		return
	}
	c.sig.Store(sig)
}

type causeKey struct{}

func WithInterruptCause(ctx context.Context, c *InterruptCause) context.Context {
	return context.WithValue(ctx, causeKey{}, c)
}

func interruptExit(ctx context.Context) int {
	c, _ := ctx.Value(causeKey{}).(*InterruptCause)
	if c != nil {
		if sig, _ := c.sig.Load().(os.Signal); sig == syscall.SIGTERM {
			return ExitTerminated
		}
	}
	return ExitInterrupt
}
