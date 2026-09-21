package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/radilabs/radichat/internal/chat"
	"github.com/radilabs/radichat/internal/config"
)

func testCfg(url string) config.Config {
	return config.Config{
		Endpoint:          url,
		Model:             "test-model",
		ContextBudget:     8192,
		GenerationReserve: 64,
	}
}

func testClient(url string) *Client {
	return New(testCfg(url), Options{
		DialTimeout:           2 * time.Second,
		TLSHandshakeTimeout:   2 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,
		IdleReadTimeout:       200 * time.Millisecond,
		MaxEventBytes:         64 << 10,
		MaxErrorBodyBytes:     1024,
		MaxStreamBytes:        1 << 20,
	})
}

func sseChunk(content, finish string) string {
	delta := map[string]any{"delta": map[string]any{}}
	if content != "" {
		delta["delta"] = map[string]any{"content": content}
	}
	if finish != "" {
		delta["finish_reason"] = finish
	} else {
		delta["finish_reason"] = nil
	}
	raw, _ := json.Marshal(map[string]any{"choices": []any{delta}})
	return "data: " + string(raw) + "\n\n"
}

func TestStreamRequestShapeAndNoAuth(t *testing.T) {
	var mu sync.Mutex
	var captured struct {
		method  string
		path    string
		headers http.Header
		body    map[string]any
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		captured.method = r.Method
		captured.path = r.URL.RequestURI()
		captured.headers = r.Header.Clone()
		dec := json.NewDecoder(r.Body)
		_ = dec.Decode(&captured.body)
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		io.WriteString(w, sseChunk("hi", ""))
		io.WriteString(w, sseChunk("", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
		fl.Flush()
	}))
	defer srv.Close()

	endpoint := srv.URL + "/v1/chat/completions"
	c := testClient(endpoint)
	msgs := []chat.Message{{Role: "user", Content: "hello"}}
	text, err := c.Stream(context.Background(), msgs, 8192, nil)
	if err != nil {
		t.Fatal(err)
	}
	if text != "hi" {
		t.Fatalf("text %q", text)
	}
	mu.Lock()
	defer mu.Unlock()
	if captured.method != http.MethodPost {
		t.Fatalf("method %s", captured.method)
	}
	if captured.path != "/v1/chat/completions" {
		t.Fatalf("path %s", captured.path)
	}
	if captured.headers.Get("Authorization") != "" {
		t.Fatal("Authorization header present")
	}
	if captured.headers.Get("Cookie") != "" {
		t.Fatal("Cookie header present")
	}
	if !strings.Contains(captured.headers.Get("Content-Type"), "application/json") {
		t.Fatalf("content-type %s", captured.headers.Get("Content-Type"))
	}
	if captured.body["model"] != "test-model" {
		t.Fatalf("model %v", captured.body["model"])
	}
	if captured.body["stream"] != true {
		t.Fatalf("stream %v", captured.body["stream"])
	}
	if captured.body["max_tokens"] != float64(64) {
		t.Fatalf("max_tokens %v", captured.body["max_tokens"])
	}
	if _, ok := captured.body["tools"]; ok {
		t.Fatal("tools present")
	}
	if _, ok := captured.body["functions"]; ok {
		t.Fatal("functions present")
	}
	if _, ok := captured.body["tool_choice"]; ok {
		t.Fatal("tool_choice present")
	}
	arr, _ := captured.body["messages"].([]any)
	if len(arr) != 1 {
		t.Fatalf("messages %#v", captured.body["messages"])
	}
}

func TestStreamIncrementalBeforeServerEnds(t *testing.T) {
	firstSeen := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		io.WriteString(w, sseChunk("hello", ""))
		fl.Flush()
		select {
		case <-firstSeen:
		case <-r.Context().Done():
			return
		}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		io.WriteString(w, sseChunk(" world", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
		fl.Flush()
	}))
	defer srv.Close()

	c := testClient(srv.URL + "/")
	var got strings.Builder
	done := make(chan error, 1)
	go func() {
		_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, func(s string) error {
			got.WriteString(s)
			if s == "hello" {
				select {
				case <-firstSeen:
				default:
					close(firstSeen)
				}
			}
			return nil
		})
		done <- err
	}()

	select {
	case <-firstSeen:
	case err := <-done:
		t.Fatalf("stream finished before first delta: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first delta")
	}
	if got.String() != "hello" {
		t.Fatalf("expected first delta before server ended, got %q", got.String())
	}
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for stream end")
	}
	if got.String() != "hello world" {
		t.Fatalf("assembled %q", got.String())
	}
}

func TestFragmentedAndCRLFAndComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Error("no hijack")
			return
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		payload := "" +
			"HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\n\r\n" +
			": comment\r\n" +
			"data: " + strings.TrimPrefix(sseChunk("A", ""), "data: ")
		payload = strings.ReplaceAll(payload, "\n\n", "\r\n\r\n")
		// Write first event with CRLF, then two events in one write, then DONE.
		second := sseChunk("B", "") + sseChunk("C", "stop") + "data: [DONE]\r\n\r\n"
		if _, err := bufrw.WriteString(payload); err != nil {
			return
		}
		if err := bufrw.Flush(); err != nil {
			return
		}
		for i := 0; i < len(payload); i++ {
			// already written as a block; additionally send second in tiny writes
		}
		for i := 0; i < len(second); i++ {
			if _, err := bufrw.Write([]byte{second[i]}); err != nil {
				return
			}
			if err := bufrw.Flush(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	c := testClient(srv.URL + "/")
	text, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err != nil {
		t.Fatal(err)
	}
	if text != "ABC" {
		t.Fatalf("got %q", text)
	}
}

func TestByteByByteJSONEvent(t *testing.T) {
	event := sseChunk("é", "stop") + "data: [DONE]\n\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "no hijack", 500)
			return
		}
		conn, bufrw, err := hj.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = bufrw.WriteString("HTTP/1.1 200 OK\r\nContent-Type: text/event-stream\r\n\r\n")
		_ = bufrw.Flush()
		for i := 0; i < len(event); i++ {
			_, _ = bufrw.Write([]byte{event[i]})
			_ = bufrw.Flush()
		}
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/")
	text, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err != nil {
		t.Fatal(err)
	}
	if text != "é" {
		t.Fatalf("got %q", text)
	}
}

func TestEmptyDeltaRoleOnlyAndLengthFinish(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":null}]}\n\n")
		io.WriteString(w, sseChunk("ok", "length"))
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/")
	text, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err != nil {
		t.Fatal(err)
	}
	if text != "ok" {
		t.Fatalf("got %q", text)
	}
}

func TestMalformedJSONEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {not json}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/")
	_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil || !strings.Contains(err.Error(), "malformed stream JSON") {
		t.Fatalf("err %v", err)
	}
}

func TestPrematureEOF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseChunk("partial", ""))
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/")
	_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil || !strings.Contains(err.Error(), "[DONE]") {
		t.Fatalf("err %v", err)
	}
}

func TestNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/")
	_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("err %v", err)
	}
}

func TestToolCallsRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"id\":\"1\"}]},\"finish_reason\":\"tool_calls\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/")
	_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil || !strings.Contains(err.Error(), "tool") {
		t.Fatalf("err %v", err)
	}
}

func TestInterruptedStream(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		io.WriteString(w, sseChunk("x", ""))
		fl.Flush()
		close(started)
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	c := testClient(srv.URL + "/")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-started
		cancel()
	}()
	_, err := c.Stream(ctx, []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil {
		t.Fatal("expected cancel")
	}
	if !errors.Is(err, context.Canceled) && !strings.Contains(err.Error(), "cancel") {
		t.Fatalf("err %v", err)
	}
}

func TestIdleStallTimesOutAndReleases(t *testing.T) {
	closed := make(chan struct{})
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		io.WriteString(w, sseChunk("hi", ""))
		fl.Flush()
		select {
		case <-r.Context().Done():
		case <-time.After(500 * time.Millisecond):
		}
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	opt := Options{
		DialTimeout:           2 * time.Second,
		TLSHandshakeTimeout:   2 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,
		IdleReadTimeout:       200 * time.Millisecond,
		MaxEventBytes:         64 << 10,
		MaxErrorBodyBytes:     1024,
		MaxStreamBytes:        1 << 20,
	}
	d := &net.Dialer{Timeout: opt.DialTimeout}
	opt.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		c, err := d.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return &closeNotifyConn{Conn: c, once: &once, closed: closed}, nil
	}
	c := New(testCfg(srv.URL+"/"), opt)
	start := time.Now()
	_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil || !strings.Contains(err.Error(), "idle timeout") {
		t.Fatalf("expected idle timeout, got %v after %s", err, time.Since(start))
	}
	if time.Since(start) > time.Second {
		t.Fatalf("timeout too slow: %v", time.Since(start))
	}
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("client connection was not closed after stall")
	}
}

type closeNotifyConn struct {
	net.Conn
	once   *sync.Once
	closed chan struct{}
}

func (c *closeNotifyConn) Close() error {
	c.once.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

func TestOngoingStreamNotTreatedAsIdle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		io.WriteString(w, sseChunk("a", ""))
		fl.Flush()
		time.Sleep(80 * time.Millisecond)
		io.WriteString(w, sseChunk("b", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
		fl.Flush()
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/")
	text, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err != nil {
		t.Fatal(err)
	}
	if text != "ab" {
		t.Fatalf("got %q", text)
	}
}

func TestRedirectsRefused(t *testing.T) {
	final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("followed redirect")
	}))
	defer final.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL, http.StatusFound)
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/")
	_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("err %v", err)
	}
}

func TestNoRetryOnFailure(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		http.Error(w, "fail", 500)
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/")
	_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if n.Load() != 1 {
		t.Fatalf("attempts %d", n.Load())
	}
}

func TestOutputBound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseChunk("abcdef", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/")
	_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 3, nil)
	if err == nil || !strings.Contains(err.Error(), "local") {
		t.Fatalf("err %v", err)
	}
}

func TestHeaderStallTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		time.Sleep(500 * time.Millisecond)
	}()
	c := New(testCfg("http://"+ln.Addr().String()+"/v1/chat/completions"), Options{
		DialTimeout:           time.Second,
		ResponseHeaderTimeout: 100 * time.Millisecond,
		IdleReadTimeout:       time.Second,
	})
	start := time.Now()
	_, err = c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil {
		t.Fatal("expected header timeout")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("too slow: %v", time.Since(start))
	}
}

func TestEndpointNotRewritten(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sseChunk("z", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/custom")
	if _, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil); err != nil {
		t.Fatal(err)
	}
	if path != "/custom" {
		t.Fatalf("path rewritten to %s", path)
	}
}

func TestFragmentedHTTPWrites(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		ev := sseChunk("hello", "") + sseChunk("!", "stop") + "data: [DONE]\n\n"
		for i := 0; i < len(ev); i++ {
			_, _ = w.Write([]byte{ev[i]})
			fl.Flush()
		}
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/")
	text, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello!" {
		t.Fatalf("got %q", text)
	}
}

func TestStreamErrorObject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"error\":{\"message\":\"bad model\"}}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/")
	_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err == nil || !strings.Contains(err.Error(), "error object") {
		t.Fatalf("err %v", err)
	}
}

func TestStreamErrorNullIsIgnored(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"error\":null,\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	c := testClient(srv.URL + "/")
	text, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err != nil {
		t.Fatal(err)
	}
	if text != "ok" {
		t.Fatalf("text %q", text)
	}
}

func TestStreamErrorNonObjectPayloadsFailClosed(t *testing.T) {
	payloads := []string{
		`{"error":[]}`,
		`{"error":false}`,
		`{"error":"nope"}`,
	}
	for _, payload := range payloads {
		t.Run(payload, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				io.WriteString(w, "data: "+payload+"\n\n")
				io.WriteString(w, "data: [DONE]\n\n")
			}))
			defer srv.Close()
			c := testClient(srv.URL + "/")
			_, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
			if err == nil || !strings.Contains(err.Error(), "malformed stream error payload") {
				t.Fatalf("err %v", err)
			}
			if strings.Contains(err.Error(), "error object in stream") {
				t.Fatalf("non-object classified as error object: %v", err)
			}
		})
	}
}

func TestOneSSEEventFragmentedAcrossIdleWindows(t *testing.T) {
	event := sseChunk("hello", "stop") + "data: [DONE]\n\n"
	parts := splitEven(event, 8)
	idle := 250 * time.Millisecond
	gap := 80 * time.Millisecond
	if time.Duration(len(parts)-1)*gap <= idle {
		t.Fatalf("test gaps must exceed idle if reset only happened per complete event")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		for i, p := range parts {
			if _, err := io.WriteString(w, p); err != nil {
				return
			}
			fl.Flush()
			if i < len(parts)-1 {
				time.Sleep(gap)
			}
		}
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	c := New(testCfg(srv.URL+"/"), Options{
		DialTimeout:           2 * time.Second,
		TLSHandshakeTimeout:   2 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,
		IdleReadTimeout:       idle,
		MaxEventBytes:         64 << 10,
		MaxErrorBodyBytes:     1024,
		MaxStreamBytes:        1 << 20,
	})
	text, err := c.Stream(context.Background(), []chat.Message{{Role: "user", Content: "x"}}, 8192, nil)
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Fatalf("text %q", text)
	}
}
