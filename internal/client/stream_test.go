package client

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestClassifyStreamError(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantSub string
		ok      bool
	}{
		{name: "omitted", raw: "", ok: true},
		{name: "json null", raw: "null", ok: true},
		{name: "object", raw: `{"message":"bad model"}`, wantSub: "error object"},
		{name: "empty object", raw: `{}`, wantSub: "error object"},
		{name: "array", raw: `[]`, wantSub: "malformed stream error payload"},
		{name: "bool", raw: `false`, wantSub: "malformed stream error payload"},
		{name: "string", raw: `"oops"`, wantSub: "malformed stream error payload"},
		{name: "number", raw: `1`, wantSub: "malformed stream error payload"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := classifyStreamError(json.RawMessage(tc.raw))
			if tc.ok {
				if err != nil {
					t.Fatalf("unexpected err %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("got %v, want substring %q", err, tc.wantSub)
			}
		})
	}
}

func TestIdleResetReaderResetsOnAnyBytes(t *testing.T) {
	var n int
	r := idleResetReader{
		r: strings.NewReader("abcdef"),
		reset: func() {
			n++
		},
	}
	buf := make([]byte, 2)
	reads := 0
	for {
		got, err := r.Read(buf)
		if got > 0 {
			reads++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if n != reads || n == 0 {
		t.Fatalf("resets=%d reads=%d", n, reads)
	}
}

func TestDecodeStreamOneEventAcrossIdleWindows(t *testing.T) {
	pr, pw := io.Pipe()
	parent := context.Background()
	streamCtx, cancel := context.WithCancel(parent)
	defer cancel()

	const idleTicks = 3
	var mu sync.Mutex
	ticks := 0
	progress := make(chan struct{}, 32)
	r := idleResetReader{
		r: pr,
		reset: func() {
			mu.Lock()
			ticks = 0
			mu.Unlock()
			select {
			case progress <- struct{}{}:
			default:
			}
		},
	}
	tick := func(n int) {
		mu.Lock()
		ticks += n
		if ticks >= idleTicks {
			cancel()
		}
		mu.Unlock()
	}

	type result struct {
		text string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		text, err := decodeStream(streamCtx, parent, r, Options{MaxEventBytes: 1 << 20, MaxStreamBytes: 1 << 20}, 8192, nil)
		done <- result{text, err}
	}()
	go func() {
		<-streamCtx.Done()
		_ = pr.Close()
	}()

	payload := sseChunk("hello", "stop") + "data: [DONE]\n\n"
	parts := splitEven(payload, 8)
	if len(parts) < 4 {
		t.Fatalf("parts %d", len(parts))
	}
	for i, p := range parts {
		if _, err := pw.Write([]byte(p)); err != nil {
			t.Fatal(err)
		}
		select {
		case <-progress:
		case <-time.After(2 * time.Second):
			t.Fatal("reader made no progress")
		}
		if i < len(parts)-1 {
			// Two ticks per gap; without per-read reset these would accumulate
			// past idleTicks before the event completed.
			tick(2)
		}
	}
	_ = pw.Close()
	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("decode: %v", res.err)
		}
		if res.text != "hello" {
			t.Fatalf("text %q", res.text)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("decode hung")
	}
}

func TestDecodeStreamStallWithoutBytesFails(t *testing.T) {
	pr, pw := io.Pipe()
	parent := context.Background()
	streamCtx, cancel := context.WithCancel(parent)
	defer cancel()

	progress := make(chan struct{}, 8)
	r := idleResetReader{
		r: pr,
		reset: func() {
			select {
			case progress <- struct{}{}:
			default:
			}
		},
	}
	type result struct {
		err error
	}
	done := make(chan result, 1)
	go func() {
		_, err := decodeStream(streamCtx, parent, r, Options{MaxEventBytes: 1 << 20, MaxStreamBytes: 1 << 20}, 8192, nil)
		done <- result{err}
	}()
	go func() {
		<-streamCtx.Done()
		_ = pr.Close()
	}()
	if _, err := pw.Write([]byte("data: {")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-progress:
	case <-time.After(2 * time.Second):
		t.Fatal("no initial progress")
	}
	cancel()
	select {
	case res := <-done:
		if res.err == nil || !strings.Contains(res.err.Error(), "idle timeout") {
			t.Fatalf("err %v", res.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stall did not fail")
	}
}

func splitEven(s string, n int) []string {
	if n < 2 || len(s) < n {
		return []string{s}
	}
	size := (len(s) + n - 1) / n
	var out []string
	for i := 0; i < len(s); i += size {
		j := i + size
		if j > len(s) {
			j = len(s)
		}
		out = append(out, s[i:j])
	}
	return out
}
