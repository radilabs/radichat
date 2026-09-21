package repl

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/radilabs/radichat/internal/client"
	"github.com/radilabs/radichat/internal/config"
)

type recordedReq struct {
	Body map[string]any
}

func chatServer(t *testing.T, handler http.HandlerFunc) (config.Config, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg := config.Config{
		Endpoint:          srv.URL + "/v1/chat/completions",
		Model:             "m",
		ContextBudget:     400,
		GenerationReserve: 40,
		SystemPrompt:      "sys",
	}
	return cfg, srv
}

func sse(content, finish string) string {
	delta := map[string]any{"delta": map[string]any{"content": content}, "finish_reason": nil}
	if finish != "" {
		delta["finish_reason"] = finish
	}
	raw, _ := json.Marshal(map[string]any{"choices": []any{delta}})
	return "data: " + string(raw) + "\n\n"
}

func TestREPLTwoTurnsClearAndQuit(t *testing.T) {
	var mu sync.Mutex
	var reqs []recordedReq
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		reqs = append(reqs, recordedReq{Body: body})
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		n := len(reqs)
		io.WriteString(w, sse("ans"+strconv.Itoa(n), "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	})
	in := strings.NewReader("hello\nagain\n/clear\nthird\n/quit\n")
	var out, errb bytes.Buffer
	code := run(context.Background(), cfg, in, &out, &errb, client.New(cfg, client.Options{}))
	if code != ExitOK {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "ans1") || !strings.Contains(out.String(), "ans2") || !strings.Contains(out.String(), "ans3") {
		t.Fatalf("stdout %q", out.String())
	}
	if !strings.Contains(errb.String(), "conversation cleared") {
		t.Fatalf("stderr %q", errb.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(reqs) != 3 {
		t.Fatalf("requests %d", len(reqs))
	}
	msgs1 := asMsgs(reqs[0].Body)
	if len(msgs1) != 2 || msgs1[0]["role"] != "sys" && msgs1[0]["content"] != "sys" {
		// system then user
	}
	if msgs1[0]["role"] != "system" || msgs1[1]["content"] != "hello" {
		t.Fatalf("turn1 %+v", msgs1)
	}
	msgs2 := asMsgs(reqs[1].Body)
	if len(msgs2) != 4 {
		t.Fatalf("turn2 should include history: %+v", msgs2)
	}
	msgs3 := asMsgs(reqs[2].Body)
	if len(msgs3) != 2 || msgs3[1]["content"] != "third" {
		t.Fatalf("after clear expected only system+user, got %+v", msgs3)
	}
}

func TestBlankAndUnknownCommandSendNoRequest(t *testing.T) {
	var n int
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		n++
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse("ok", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	})
	in := strings.NewReader("\n   \n/nope\n/quit\n")
	var out, errb bytes.Buffer
	code := run(context.Background(), cfg, in, &out, &errb, client.New(cfg, client.Options{}))
	if code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if n != 0 {
		t.Fatalf("sent %d requests", n)
	}
	if !strings.Contains(errb.String(), "unknown command") {
		t.Fatalf("stderr %q", errb.String())
	}
}

func TestOversizedMessageNoRequest(t *testing.T) {
	var n int
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		n++
	})
	cfg.ContextBudget = 80
	cfg.GenerationReserve = 50
	in := strings.NewReader(strings.Repeat("x", 200) + "\n/quit\n")
	var out, errb bytes.Buffer
	code := run(context.Background(), cfg, in, &out, &errb, client.New(cfg, client.Options{}))
	if code != ExitOK {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	if n != 0 {
		t.Fatal("oversized prompt contacted network")
	}
	if !strings.Contains(errb.String(), "does not fit") {
		t.Fatalf("stderr %q", errb.String())
	}
}

func TestREPLTrimNotice(t *testing.T) {
	var n int
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		n++
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse(strings.Repeat("a", 10), "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	})
	cfg.ContextBudget = 180
	cfg.GenerationReserve = 30
	cfg.SystemPrompt = ""
	in := strings.NewReader("aaaaaaaaaa\naaaaaaaaaa\naaaaaaaaaa\naaaaaaaaaa\n/quit\n")
	var out, errb bytes.Buffer
	code := run(context.Background(), cfg, in, &out, &errb, client.New(cfg, client.Options{}))
	if code != ExitOK {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	if n < 2 {
		t.Fatalf("requests %d", n)
	}
	if !strings.Contains(errb.String(), "dropped") {
		t.Fatalf("expected trim notice, stderr %q", errb.String())
	}
}

func TestFailedTurnRollbackThenRecover(t *testing.T) {
	var mu sync.Mutex
	var reqs [][]map[string]any
	n := 0
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		reqs = append(reqs, asMsgs(body))
		n++
		cur := n
		mu.Unlock()
		if cur == 1 {
			http.Error(w, "boom", 500)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse("ok", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	})
	in := strings.NewReader("first\nsecond\n/quit\n")
	var out, errb bytes.Buffer
	code := run(context.Background(), cfg, in, &out, &errb, client.New(cfg, client.Options{}))
	if code != ExitOK {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "500") {
		t.Fatalf("stderr %q", errb.String())
	}
	if !strings.Contains(out.String(), "ok") {
		t.Fatalf("stdout %q", out.String())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(reqs) != 2 {
		t.Fatalf("reqs %d", len(reqs))
	}
	if len(reqs[1]) != 2 || reqs[1][1]["content"] != "second" {
		t.Fatalf("second request should not include failed first turn: %+v", reqs[1])
	}
}

func TestEOFExits(t *testing.T) {
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse("z", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	})
	in := strings.NewReader("hi")
	var out, errb bytes.Buffer
	code := run(context.Background(), cfg, in, &out, &errb, client.New(cfg, client.Options{}))
	if code != ExitOK {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
}

func TestInterruptDuringStreamDiscardsTurn(t *testing.T) {
	block := make(chan struct{})
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		io.WriteString(w, sse("partial", ""))
		fl.Flush()
		select {
		case <-block:
		case <-r.Context().Done():
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	inR, inW := io.Pipe()
	var out, errb bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- run(ctx, cfg, inR, &out, &errb, client.New(cfg, client.Options{
			IdleReadTimeout:       5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
		}))
	}()
	if _, err := io.WriteString(inW, "hello\n"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case code := <-done:
		if code != ExitInterrupt {
			t.Fatalf("exit %d stderr %s", code, errb.String())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("hang on interrupt")
	}
	close(block)
	_ = inW.Close()
}

func TestNoPersistenceArtifacts(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	cfg, _ := chatServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, sse("z", "stop"))
		io.WriteString(w, "data: [DONE]\n\n")
	})
	in := strings.NewReader("hi\n/quit\n")
	var out, errb bytes.Buffer
	if code := run(context.Background(), cfg, in, &out, &errb, client.New(cfg, client.Options{})); code != ExitOK {
		t.Fatalf("exit %d %s", code, errb.String())
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if strings.Contains(strings.ToLower(name), "hist") || strings.HasSuffix(name, ".log") || name == "radichat.json" {
			t.Fatalf("unexpected artifact %s", name)
		}
	}
}

func asMsgs(body map[string]any) []map[string]any {
	raw, _ := body["messages"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, m := range raw {
		mm, _ := m.(map[string]any)
		out = append(out, mm)
	}
	return out
}
