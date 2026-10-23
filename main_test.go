package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/radilabs/radichat/internal/config"
)

func TestPromptFitStartup(t *testing.T) {
	const (
		budget  = int64(200)
		reserve = int64(10)
	)
	// system + empty user + reserve: 32 + (len+32) + 32 + reserve <= budget.
	exact := strings.Repeat("p", int(budget-reserve-96))
	over := exact + "Q"

	t.Run("no prompt", func(t *testing.T) {
		assertPromptFit(t, budget, reserve, "", "", "/quit\n", 0, 0, "")
	})
	t.Run("empty inline", func(t *testing.T) {
		assertPromptFit(t, 100, 10, "", "empty", "/quit\n", 0, 0, "")
	})
	t.Run("inline exact", func(t *testing.T) {
		assertPromptFit(t, budget, reserve, exact, "inline", "/quit\n", 0, 0, "")
	})
	t.Run("inline over", func(t *testing.T) {
		assertPromptFit(t, budget, reserve, over, "inline", "/quit\n", 1, 0, "system prompt does not fit")
	})
	t.Run("file exact", func(t *testing.T) {
		assertPromptFit(t, budget, reserve, exact, "file", "/quit\n", 0, 0, "")
	})
	t.Run("file over", func(t *testing.T) {
		assertPromptFit(t, budget, reserve, over, "file", "/quit\n", 1, 0, "system prompt does not fit")
	})
}

func assertPromptFit(t *testing.T, budget, reserve int64, prompt, mode, input string, wantCode, wantHits int, wantErr string) {
	t.Helper()
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"x\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "radichat.json")
	switch mode {
	case "", "inline":
		cfg := map[string]any{
			"endpoint":           srv.URL + "/v1/chat/completions",
			"model":              "m",
			"context_budget":     budget,
			"generation_reserve": reserve,
		}
		if mode == "inline" {
			cfg["system_prompt"] = prompt
		}
		raw, err := json.Marshal(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(cfgPath, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	case "empty":
		raw := fmt.Sprintf(`{"endpoint":%q,"model":"m","context_budget":%d,"generation_reserve":%d,"system_prompt":""}`, srv.URL+"/v1/chat/completions", budget, reserve)
		if err := os.WriteFile(cfgPath, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
	case "file":
		if err := os.WriteFile(filepath.Join(dir, "prompt.txt"), []byte(prompt), 0o600); err != nil {
			t.Fatal(err)
		}
		raw := fmt.Sprintf(`{"endpoint":%q,"model":"m","context_budget":%d,"generation_reserve":%d,"system_prompt_file":"prompt.txt"}`, srv.URL+"/v1/chat/completions", budget, reserve)
		if err := os.WriteFile(cfgPath, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("mode %q", mode)
	}
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", cfgPath}, strings.NewReader(input), &out, &errb)
	if code != wantCode {
		t.Fatalf("exit %d want %d stderr %q stdout %q", code, wantCode, errb.String(), out.String())
	}
	if hits != wantHits {
		t.Fatalf("requests %d want %d", hits, wantHits)
	}
	if wantErr != "" && !strings.Contains(errb.String(), wantErr) {
		t.Fatalf("stderr %q does not contain %q", errb.String(), wantErr)
	}
	if prompt != "" && strings.Contains(errb.String(), prompt) {
		t.Fatalf("stderr leaked prompt: %q", errb.String())
	}
	if prompt != "" && strings.Contains(out.String(), prompt) {
		t.Fatalf("stdout leaked prompt: %q", out.String())
	}
}

func TestHelpListsInput(t *testing.T) {
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-help"}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "-input") {
		t.Fatalf("help %q", out.String())
	}
	if !strings.Contains(out.String(), "./-") && !strings.Contains(out.String(), "named -") {
		t.Fatalf("help should say a file named - is ./-: %q", out.String())
	}
}

func TestCLIInputFile(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		got = append(got, lastContent(msgs(body)))
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"pong\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "radichat.json")
	writeCfgAt(t, cfgPath, srv.URL+"/v1/chat/completions", 400, 40, "")
	inputPath := filepath.Join(dir, "in.txt")
	if err := os.WriteFile(inputPath, []byte("from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	devnull, err := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { devnull.Close() })
	var errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", cfgPath, "-input", inputPath}, strings.NewReader("from-stdin\n"), devnull, &errb)
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	if strings.Contains(errb.String(), "> ") {
		t.Fatalf("file input prompted: %q", errb.String())
	}
	if len(got) != 1 || got[0] != "from-file" {
		t.Fatalf("requests %v", got)
	}
}

func TestCLIInputMissingAndDirectory(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request issued")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "radichat.json")
	writeCfgAt(t, cfgPath, srv.URL+"/v1/chat/completions", 400, 40, "")
	outPath := filepath.Join(dir, "out.jsonl")

	t.Run("missing", func(t *testing.T) {
		var out, errb bytes.Buffer
		code := run(context.Background(), []string{"radichat", "-config", cfgPath, "-input", filepath.Join(dir, "missing.txt"), "-output", outPath}, strings.NewReader("hi\n"), &out, &errb)
		if code != 1 {
			t.Fatalf("exit %d stderr %q", code, errb.String())
		}
		if !strings.Contains(errb.String(), "not found") {
			t.Fatalf("stderr %q", errb.String())
		}
		if _, err := os.Stat(outPath); !os.IsNotExist(err) {
			t.Fatalf("output created: %v", err)
		}
	})
	t.Run("directory", func(t *testing.T) {
		var out, errb bytes.Buffer
		code := run(context.Background(), []string{"radichat", "-config", cfgPath, "-input", dir, "-output", outPath}, strings.NewReader("hi\n"), &out, &errb)
		if code != 1 {
			t.Fatalf("exit %d stderr %q", code, errb.String())
		}
		if !strings.Contains(errb.String(), "directory") {
			t.Fatalf("stderr %q", errb.String())
		}
		if _, err := os.Stat(outPath); !os.IsNotExist(err) {
			t.Fatalf("output created: %v", err)
		}
	})
}

func TestCLIInputDashIsLocalFile(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		got = append(got, lastContent(msgs(body)))
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "radichat.json")
	writeCfgAt(t, cfgPath, srv.URL+"/v1/chat/completions", 400, 40, "")
	if err := os.WriteFile(filepath.Join(dir, "-"), []byte("dash-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", cfgPath, "-input", "-"}, strings.NewReader("from-stdin\n"), &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d stderr %s stdout %s", code, errb.String(), out.String())
	}
	if len(got) != 1 || got[0] != "dash-file" {
		t.Fatalf("requests %v stdout %q", got, out.String())
	}
}

func TestCLIInputFIFO(t *testing.T) {
	var got []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		got = append(got, lastContent(msgs(body)))
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "radichat.json")
	writeCfgAt(t, cfgPath, srv.URL+"/v1/chat/completions", 400, 40, "")
	fifo := filepath.Join(dir, "in.fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	errc := make(chan error, 1)
	go func() {
		f, err := os.OpenFile(fifo, os.O_WRONLY, 0)
		if err != nil {
			errc <- err
			return
		}
		defer f.Close()
		_, err = io.WriteString(f, "fifo-line\n")
		errc <- err
	}()
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", cfgPath, "-input", fifo}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d stderr %s stdout %s", code, errb.String(), out.String())
	}
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "fifo-line" {
		t.Fatalf("requests %v stdout %q", got, out.String())
	}
}

func TestHelp(t *testing.T) {
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-help"}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "-config") {
		t.Fatalf("help %q", out.String())
	}
}

func TestMissingConfig(t *testing.T) {
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", filepath.Join(t.TempDir(), "nope.json")}, strings.NewReader(""), &out, &errb)
	if code == 0 {
		t.Fatal("expected nonzero")
	}
	if !strings.Contains(errb.String(), "not found") {
		t.Fatalf("stderr %q", errb.String())
	}
}

func TestInvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte(`{"model":"m"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", path}, strings.NewReader(""), &out, &errb)
	if code == 0 {
		t.Fatal("expected nonzero")
	}
	if errb.Len() == 0 {
		t.Fatal("expected stderr")
	}
}

func TestRunWithConfigAndEOF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("auth header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"pong\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	path := writeCfg(t, srv.URL+"/v1/chat/completions", 8192, 1024, "sys")
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", path}, strings.NewReader("ping\n"), &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "pong") {
		t.Fatalf("stdout %q", out.String())
	}
}

func TestBinaryBlackBox(t *testing.T) {
	bin := buildBin(t)
	var mu sync.Mutex
	var bodies []map[string]any
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("unexpected auth")
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		bodies = append(bodies, body)
		n++
		cur := n
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, fmt.Sprintf("data: {\"choices\":[{\"delta\":{\"content\":\"t%d\"},\"finish_reason\":\"stop\"}]}\n\n", cur))
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "radichat.json")
	writeCfgAt(t, cfgPath, srv.URL+"/v1/chat/completions", 300, 40, "sys")

	cmd := exec.Command(bin, "-config", cfgPath)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader("hello\nagain\n/clear\nthird\n/quit\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	s := string(out)
	if !strings.Contains(s, "t1") || !strings.Contains(s, "t2") || !strings.Contains(s, "t3") {
		t.Fatalf("output %q", s)
	}
	mu.Lock()
	if len(bodies) != 3 {
		mu.Unlock()
		t.Fatalf("requests %d", len(bodies))
	}
	m1 := msgs(bodies[0])
	m2 := msgs(bodies[1])
	m3 := msgs(bodies[2])
	mu.Unlock()
	if lastContent(m1) != "hello" || lastContent(m2) != "again" || lastContent(m3) != "third" {
		t.Fatalf("contents %+v %+v %+v", m1, m2, m3)
	}
	if len(m2) <= len(m1) {
		t.Fatalf("expected history on turn 2: %d vs %d", len(m2), len(m1))
	}
	if len(m3) != 2 {
		t.Fatalf("after clear: %+v", m3)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "radichat.json" {
			t.Fatalf("artifact %s", e.Name())
		}
	}

	srv.CloseClientConnections()

	cmd2 := exec.Command(bin, "-config", cfgPath)
	cmd2.Dir = dir
	cmd2.Stdin = strings.NewReader("fresh\n/quit\n")
	out2, err := cmd2.CombinedOutput()
	if err != nil {
		t.Fatalf("second process: %v\n%s", err, out2)
	}
	if !strings.Contains(string(out2), "t4") {
		t.Fatalf("second process output %q", out2)
	}
	mu.Lock()
	m4 := msgs(bodies[len(bodies)-1])
	mu.Unlock()
	if lastContent(m4) != "fresh" || len(m4) != 2 {
		t.Fatalf("second process reused history: %+v", m4)
	}
}

func TestBinaryFailedTurnRecover(t *testing.T) {
	bin := buildBin(t)
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			http.Error(w, "nope", 503)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	cfgPath := writeCfg(t, srv.URL+"/v1/chat/completions", 400, 40, "")
	cmd := exec.Command(bin, "-config", cfgPath)
	cmd.Stdin = strings.NewReader("one\ntwo\n/quit\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(string(out), "503") || !strings.Contains(string(out), "ok") {
		t.Fatalf("output %q", out)
	}
}

func TestBinaryTrimming(t *testing.T) {
	bin := buildBin(t)
	var mu sync.Mutex
	var lengths []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		lengths = append(lengths, len(msgs(body)))
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"aaaaaaaaaa\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	cfgPath := writeCfg(t, srv.URL+"/v1/chat/completions", 180, 30, "")
	cmd := exec.Command(bin, "-config", cfgPath)
	cmd.Stdin = strings.NewReader("aaaaaaaaaa\naaaaaaaaaa\naaaaaaaaaa\naaaaaaaaaa\n/quit\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(string(out), "dropped") {
		t.Fatalf("expected trim notice in %q", out)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(lengths) < 2 {
		t.Fatalf("lengths %v", lengths)
	}
	grewThenCapped := false
	for i := 1; i < len(lengths); i++ {
		if lengths[i] <= lengths[i-1] {
			grewThenCapped = true
		}
	}
	if !grewThenCapped {
		t.Fatalf("expected trimming to cap request size, lengths=%v", lengths)
	}
}

func TestBinaryCtrlCDuringStream(t *testing.T) {
	bin := buildBin(t)
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\n")
		fl.Flush()
		close(started)
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	cfgPath := writeCfg(t, srv.URL+"/v1/chat/completions", 400, 40, "")
	cmd := exec.Command(bin, "-config", cfgPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	if _, err := io.WriteString(stdin, "hello\n"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("server never saw request")
	}
	if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				if ee.ExitCode() != 130 && ee.ExitCode() != 0 {
					t.Fatalf("exit %d", ee.ExitCode())
				}
			} else {
				t.Fatal(err)
			}
		}
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("hang after SIGINT")
	}
}

func TestConfigurableFieldsWithoutRebuild(t *testing.T) {
	bin := buildBin(t)
	var mu sync.Mutex
	var models []string
	var maxTok []any
	var sys []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		models = append(models, fmt.Sprint(body["model"]))
		maxTok = append(maxTok, body["max_tokens"])
		ms := msgs(body)
		if len(ms) > 0 && fmt.Sprint(ms[0]["role"]) == "system" {
			sys = append(sys, fmt.Sprint(ms[0]["content"]))
		} else {
			sys = append(sys, "")
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"x\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	a := filepath.Join(t.TempDir(), "a.json")
	writeCfgAt(t, a, srv.URL+"/v1/chat/completions", 400, 40, "one")
	// overwrite model via second file
	bdir := t.TempDir()
	b := filepath.Join(bdir, "b.json")
	raw := fmt.Sprintf(`{"endpoint":%q,"model":"other","context_budget":500,"generation_reserve":80,"system_prompt":"two"}`, srv.URL+"/v1/chat/completions")
	if err := os.WriteFile(b, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{a, b} {
		cmd := exec.Command(bin, "-config", path)
		cmd.Stdin = strings.NewReader("hi\n/quit\n")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", path, err, out)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(models) != 2 || models[0] == models[1] {
		t.Fatalf("models %v", models)
	}
	if fmt.Sprint(maxTok[0]) == fmt.Sprint(maxTok[1]) {
		t.Fatalf("reserve/max_tokens did not change: %v", maxTok)
	}
	if sys[0] != "one" || sys[1] != "two" {
		t.Fatalf("system %v", sys)
	}
}

func buildBin(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "radichat")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func writeCfg(t *testing.T, endpoint string, budget, reserve int64, sys string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "radichat.json")
	writeCfgAt(t, path, endpoint, budget, reserve, sys)
	return path
}

func writeCfgAt(t *testing.T, path, endpoint string, budget, reserve int64, sys string) {
	t.Helper()
	cfg := map[string]any{
		"endpoint":           endpoint,
		"model":              "m",
		"context_budget":     budget,
		"generation_reserve": reserve,
	}
	if sys != "" {
		cfg["system_prompt"] = sys
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Parse(path, raw); err != nil {
		t.Fatal(err)
	}
}

func msgs(body map[string]any) []map[string]any {
	raw, _ := body["messages"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, m := range raw {
		mm, _ := m.(map[string]any)
		out = append(out, mm)
	}
	return out
}

func lastContent(ms []map[string]any) string {
	if len(ms) == 0 {
		return ""
	}
	return fmt.Sprint(ms[len(ms)-1]["content"])
}
