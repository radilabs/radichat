package repl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/radilabs/radichat/internal/client"
)

func TestJSONLREPLSessionRecords(t *testing.T) {
	var mu sync.Mutex
	var reqs int
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reqs++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse("pong", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	})
	in := strings.NewReader("\nhello\n\n /nope \n/clear\n/quit\nignored\n")
	var out, errb, rec bytes.Buffer
	code := RunWithOptions(context.Background(), cfg, in, &out, &errb, Options{
		Client:  client.New(cfg, client.Options{}),
		Records: NewJSONLWriter(&rec),
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	rows := decodeRecords(t, rec.Bytes())
	if len(rows) != 4 {
		t.Fatalf("records %d %s", len(rows), rec.String())
	}
	if rows[0]["line"] != 2.0 || rows[0]["kind"] != "turn" || rows[0]["status"] != "ok" {
		t.Fatalf("turn %+v", rows[0])
	}
	if rows[0]["input"] != "hello" || rows[0]["output"] != "pong" || rows[0]["finish_reason"] != "stop" {
		t.Fatalf("turn fields %+v", rows[0])
	}
	if rows[1]["line"] != 4.0 || rows[1]["kind"] != "command_error" || rows[1]["input"] != " /nope " {
		t.Fatalf("command %+v", rows[1])
	}
	if _, ok := rows[1]["status"]; ok {
		t.Fatalf("command status present %+v", rows[1])
	}
	if rows[2]["kind"] != "clear" || rows[2]["line"] != 5.0 {
		t.Fatalf("clear %+v", rows[2])
	}
	if rows[3]["kind"] != "quit" || rows[3]["line"] != 6.0 {
		t.Fatalf("quit %+v", rows[3])
	}
	mu.Lock()
	defer mu.Unlock()
	if reqs != 1 {
		t.Fatalf("requests %d", reqs)
	}
}

func TestJSONLREPLEOFWithoutNewline(t *testing.T) {
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse("z", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	})
	var out, errb, rec bytes.Buffer
	code := RunWithOptions(context.Background(), cfg, strings.NewReader("hello"), &out, &errb, Options{
		Client:  client.New(cfg, client.Options{}),
		Records: NewJSONLWriter(&rec),
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	rows := decodeRecords(t, rec.Bytes())
	if len(rows) != 1 || rows[0]["line"] != 1.0 || rows[0]["input"] != "hello" || rows[0]["output"] != "z" {
		t.Fatalf("rows %+v", rows)
	}
}

func TestJSONLREPLLocalAndStreamErrors(t *testing.T) {
	var mu sync.Mutex
	var bodies [][]map[string]any
	n := 0
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		bodies = append(bodies, asMsgs(body))
		n++
		cur := n
		mu.Unlock()
		if cur == 1 {
			http.Error(w, "boom", 500)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse("SECRETPARTIAL", ""))
		// end before [DONE]
	})
	cfg.ContextBudget = 200
	cfg.GenerationReserve = 40
	cfg.SystemPrompt = ""
	var out, errb, rec bytes.Buffer
	input := strings.Repeat("x", 200) + "\n" + "first\n" + "second\n/quit\n"
	code := RunWithOptions(context.Background(), cfg, strings.NewReader(input), &out, &errb, Options{
		Client:  client.New(cfg, client.Options{}),
		Records: NewJSONLWriter(&rec),
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr %s json %s", code, errb.String(), rec.String())
	}
	if strings.Contains(rec.String(), "SECRETPARTIAL") {
		t.Fatalf("partial leaked into records: %s", rec.String())
	}
	rows := decodeRecords(t, rec.Bytes())
	if len(rows) != 4 {
		t.Fatalf("rows %d %s", len(rows), rec.String())
	}
	if rows[0]["status"] != "error" || rows[0]["sent"] != false {
		t.Fatalf("preflight %+v", rows[0])
	}
	if _, ok := rows[0]["output"]; ok {
		t.Fatalf("preflight output %+v", rows[0])
	}
	if rows[1]["status"] != "error" || rows[1]["sent"] != true || rows[1]["input"] != "first" {
		t.Fatalf("http %+v", rows[1])
	}
	if _, ok := rows[1]["output"]; ok {
		t.Fatalf("http output %+v", rows[1])
	}
	if rows[2]["status"] != "error" || rows[2]["sent"] != true {
		t.Fatalf("partial %+v", rows[2])
	}
	if _, ok := rows[2]["output"]; ok {
		t.Fatalf("partial output %+v", rows[2])
	}
	if rows[3]["kind"] != "quit" {
		t.Fatalf("quit %+v", rows[3])
	}
	mu.Lock()
	defer mu.Unlock()
	if len(bodies) != 2 {
		t.Fatalf("requests %d", len(bodies))
	}
	if len(bodies[1]) != 1 || bodies[1][0]["content"] != "second" {
		t.Fatalf("failed turns were committed: %+v", bodies[1])
	}
}

func TestJSONLREPLOverlongOmitsInput(t *testing.T) {
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("overlong line contacted the endpoint")
	})
	cfg.ContextBudget = 5000
	cfg.GenerationReserve = 10
	var out, errb, rec bytes.Buffer
	line := strings.Repeat("z", 5001) + "\n/quit\n"
	code := RunWithOptions(context.Background(), cfg, strings.NewReader(line), &out, &errb, Options{
		Client:  client.New(cfg, client.Options{}),
		Records: NewJSONLWriter(&rec),
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	rows := decodeRecords(t, rec.Bytes())
	if len(rows) != 2 {
		t.Fatalf("rows %+v", rows)
	}
	if rows[0]["sent"] != false || rows[0]["error"] != "line too long" {
		t.Fatalf("overlong %+v", rows[0])
	}
	if _, ok := rows[0]["input"]; ok {
		t.Fatalf("overlong included input")
	}
}

func TestRecordSurvivesTrimAndClear(t *testing.T) {
	var mu sync.Mutex
	var last []map[string]any
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		msgs := asMsgs(body)
		mu.Lock()
		last = msgs
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse(strings.Repeat("a", 10), "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	})
	cfg.ContextBudget = 180
	cfg.GenerationReserve = 30
	cfg.SystemPrompt = "sys"
	in := strings.NewReader("aaaaaaaaaa\naaaaaaaaaa\naaaaaaaaaa\naaaaaaaaaa\n/clear\nzzzzzzzzzz\n/quit\n")
	var out, errb, rec bytes.Buffer
	code := RunWithOptions(context.Background(), cfg, in, &out, &errb, Options{
		Client:  client.New(cfg, client.Options{}),
		Records: NewJSONLWriter(&rec),
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "dropped") {
		t.Fatalf("expected trim, stderr %q", errb.String())
	}
	rows := decodeRecords(t, rec.Bytes())
	var beforeClear []map[string]any
	seenClear := false
	for _, row := range rows {
		if row["kind"] == "clear" {
			seenClear = true
			continue
		}
		if !seenClear && row["kind"] == "turn" && row["status"] == "ok" {
			beforeClear = append(beforeClear, row)
		}
	}
	if !seenClear {
		t.Fatalf("clear record missing: %s", rec.String())
	}
	if len(beforeClear) != 4 {
		t.Fatalf("records before clear: %+v", beforeClear)
	}
	if beforeClear[0]["trimmed_pairs"] != 0.0 || beforeClear[0]["output"] != strings.Repeat("a", 10) {
		t.Fatalf("first record %+v", beforeClear[0])
	}
	trimmed := false
	for _, row := range beforeClear[1:] {
		if row["output"] != strings.Repeat("a", 10) {
			t.Fatalf("earlier output dropped: %+v", row)
		}
		if tp, _ := row["trimmed_pairs"].(float64); tp > 0 {
			trimmed = true
		}
	}
	if !trimmed {
		t.Fatalf("later records did not report trimmed pairs: %s", rec.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(last) != 2 || last[0]["content"] != "sys" || last[1]["content"] != "zzzzzzzzzz" {
		t.Fatalf("after clear %+v", last)
	}
}

func TestJSONLREPLWriteFailureStopsRequests(t *testing.T) {
	var n int
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		n++
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse("ok", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	})
	var out, errb bytes.Buffer
	code := RunWithOptions(context.Background(), cfg, strings.NewReader("one\ntwo\n"), &out, &errb, Options{
		Client:  client.New(cfg, client.Options{}),
		Records: failWriter{},
	})
	if code != ExitError {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	if n != 1 {
		t.Fatalf("requests %d", n)
	}
}

func TestJSONLStdoutSuppressesDeltas(t *testing.T) {
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse("pong", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	})
	var stdout, errb bytes.Buffer
	code := RunWithOptions(context.Background(), cfg, strings.NewReader("hi\n"), &stdout, &errb, Options{
		Client:      client.New(cfg, client.Options{}),
		Records:     NewJSONLWriter(&stdout),
		JSONLStdout: true,
	})
	if code != ExitOK {
		t.Fatalf("exit %d stderr %s stdout %s", code, errb.String(), stdout.String())
	}
	rows := decodeRecords(t, stdout.Bytes())
	if len(rows) != 1 || rows[0]["output"] != "pong" {
		t.Fatalf("stdout %s", stdout.String())
	}
	if strings.Contains(errb.String(), "pong") {
		t.Fatalf("delta on stderr %q", errb.String())
	}
}

func TestJSONLInterruptRecordOmitsPartial(t *testing.T) {
	// The handler must be able to return without observing client disconnect.
	// net/http cancels the server request context when ServeHTTP returns, and
	// httptest.Server.Close waits for that return, so a handler blocked only on
	// r.Context() deadlocks Close while the accepted connection is still active.
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseHandler := func() { releaseOnce.Do(func() { close(release) }) }
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		io.WriteString(w, sse("SECRETPARTIAL", ""))
		fl.Flush()
		close(entered)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	// Registered after chatServer so it runs before httptest.Server.Close.
	t.Cleanup(releaseHandler)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	inR, inW := io.Pipe()
	t.Cleanup(func() { _ = inW.Close() })
	var out, errb, rec bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- RunWithOptions(ctx, cfg, inR, &out, &errb, Options{
			Client: client.New(cfg, client.Options{
				IdleReadTimeout:       5 * time.Second,
				ResponseHeaderTimeout: 5 * time.Second,
			}),
			Records: NewJSONLWriter(&rec),
		})
	}()
	if _, err := io.WriteString(inW, "hello\n"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("server never accepted the in-flight turn")
	}
	cancel()
	select {
	case code := <-done:
		if code != ExitInterrupt {
			t.Fatalf("exit %d stderr %s", code, errb.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("hang")
	}
	if strings.Contains(rec.String(), "SECRETPARTIAL") {
		t.Fatalf("partial in record %s", rec.String())
	}
	rows := decodeRecords(t, rec.Bytes())
	if len(rows) != 1 || rows[0]["status"] != "interrupted" || rows[0]["input"] != "hello" {
		t.Fatalf("rows %+v %s", rows, rec.String())
	}
	if _, ok := rows[0]["output"]; ok {
		t.Fatalf("interrupted output %+v", rows[0])
	}
	releaseHandler()
}

func TestInterruptWriteFailureKeepsSignalExit(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseHandler := func() { releaseOnce.Do(func() { close(release) }) }
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		io.WriteString(w, sse("SECRETPARTIAL", ""))
		fl.Flush()
		close(entered)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	t.Cleanup(releaseHandler)
	cause := NewInterruptCause()
	ctx, cancel := context.WithCancel(WithInterruptCause(context.Background(), cause))
	t.Cleanup(cancel)
	inR, inW := io.Pipe()
	t.Cleanup(func() { _ = inW.Close() })
	var out, errb bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- RunWithOptions(ctx, cfg, inR, &out, &errb, Options{
			Client: client.New(cfg, client.Options{
				IdleReadTimeout:       5 * time.Second,
				ResponseHeaderTimeout: 5 * time.Second,
			}),
			Records: failWriter{},
		})
	}()
	if _, err := io.WriteString(inW, "hello\n"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("server never accepted the in-flight turn")
	}
	cause.Store(syscall.SIGTERM)
	cancel()
	select {
	case code := <-done:
		if code != ExitTerminated {
			t.Fatalf("exit %d stderr %s", code, errb.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("hang")
	}
	if strings.Contains(out.String(), "SECRETPARTIAL") && strings.Contains(out.String(), `"status":"ok"`) {
		t.Fatalf("partial recorded as success: %s", out.String())
	}
	releaseHandler()
}

type failWriter struct{}

func (failWriter) WriteRecord(Record) error { return errors.New("disk full") }

func decodeRecords(t *testing.T, b []byte) []map[string]any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(b))
	var rows []map[string]any
	for {
		var row map[string]any
		err := dec.Decode(&row)
		if errors.Is(err, io.EOF) {
			return rows
		}
		if err != nil {
			t.Fatalf("decode %v in %s", err, b)
		}
		rows = append(rows, row)
	}
}
