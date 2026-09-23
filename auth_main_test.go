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
	"sync/atomic"
	"testing"
)

const phase2Sentinel = "phase2-test-sentinel-aa11"

func writeAuthCfgAt(t *testing.T, path, endpoint, envName string) {
	t.Helper()
	raw := fmt.Sprintf(`{"endpoint":%q,"model":"m","context_budget":400,"generation_reserve":40,"bearer_token_env":%q}`, endpoint, envName)
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRunMissingBearerEnvDoesNotContactNetwork(t *testing.T) {
	var hit atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Store(true)
		t.Error("network reached")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	path := filepath.Join(t.TempDir(), "radichat.json")
	writeAuthCfgAt(t, path, srv.URL+"/v1/chat/completions", "RADICHAT_PHASE2_TEST_UNSET")
	_ = os.Unsetenv("RADICHAT_PHASE2_TEST_UNSET")
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", path}, strings.NewReader("hi\n"), &out, &errb)
	if code == 0 {
		t.Fatal("expected nonzero")
	}
	if hit.Load() {
		t.Fatal("contacted network with missing token")
	}
	msg := errb.String()
	if !strings.Contains(msg, "unset") {
		t.Fatalf("stderr %q", msg)
	}
	if strings.Contains(msg, phase2Sentinel) {
		t.Fatal("stderr disclosed sentinel")
	}
}

func TestRunEmptyBearerEnvDoesNotContactNetwork(t *testing.T) {
	var hit atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Store(true)
	}))
	t.Cleanup(srv.Close)
	path := filepath.Join(t.TempDir(), "radichat.json")
	writeAuthCfgAt(t, path, srv.URL+"/v1/chat/completions", "RADICHAT_PHASE2_TEST_EMPTY")
	t.Setenv("RADICHAT_PHASE2_TEST_EMPTY", "")
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", path}, strings.NewReader("hi\n"), &out, &errb)
	if code == 0 {
		t.Fatal("expected nonzero")
	}
	if hit.Load() {
		t.Fatal("contacted network with empty token")
	}
	if !strings.Contains(errb.String(), "empty") {
		t.Fatalf("stderr %q", errb.String())
	}
	if strings.Contains(errb.String(), phase2Sentinel) {
		t.Fatal("stderr disclosed sentinel")
	}
}

func TestRunMalformedBearerDoesNotContactNetwork(t *testing.T) {
	var hit atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Store(true)
	}))
	t.Cleanup(srv.Close)
	path := filepath.Join(t.TempDir(), "radichat.json")
	writeAuthCfgAt(t, path, srv.URL+"/v1/chat/completions", "RADICHAT_PHASE2_TEST_LF")
	t.Setenv("RADICHAT_PHASE2_TEST_LF", "abc\ndef")
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", path}, strings.NewReader("hi\n"), &out, &errb)
	if code == 0 {
		t.Fatal("expected nonzero")
	}
	if hit.Load() {
		t.Fatal("contacted network with malformed token")
	}
	if strings.Contains(errb.String(), "abc") || strings.Contains(errb.String(), "def") {
		t.Fatal("stderr disclosed malformed token")
	}
}

func TestRunAuthenticatedChat(t *testing.T) {
	t.Setenv("RADICHAT_PHASE2_TEST_TOKEN", phase2Sentinel)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+phase2Sentinel {
			t.Error("missing or unexpected Authorization header")
			http.Error(w, "nope", 401)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"pong\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	path := filepath.Join(t.TempDir(), "radichat.json")
	writeAuthCfgAt(t, path, srv.URL+"/v1/chat/completions", "RADICHAT_PHASE2_TEST_TOKEN")
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", path}, strings.NewReader("ping\n"), &out, &errb)
	if strings.Contains(out.String(), phase2Sentinel) || strings.Contains(errb.String(), phase2Sentinel) {
		t.Fatal("output disclosed sentinel")
	}
	if code != 0 {
		t.Fatal("authenticated chat exited nonzero")
	}
	if !strings.Contains(out.String(), "pong") {
		t.Fatal("expected streamed assistant text")
	}
}

func TestBinaryAuthAndNoAuthBlackBox(t *testing.T) {
	bin := buildBin(t)
	var auths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auths = append(auths, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})

	openPath := writeCfg(t, srv.URL+"/v1/chat/completions", 400, 40, "")
	cmd := exec.Command(bin, "-config", openPath)
	cmd.Stdin = strings.NewReader("hello\n/quit\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("no-auth: %v\n%s", err, out)
	}

	authPath := filepath.Join(t.TempDir(), "auth.json")
	writeAuthCfgAt(t, authPath, srv.URL+"/v1/chat/completions", "RADICHAT_PHASE2_TEST_TOKEN")
	cmd = exec.Command(bin, "-config", authPath)
	cmd.Env = append(environWithout("RADICHAT_PHASE2_TEST_TOKEN"), "RADICHAT_PHASE2_TEST_TOKEN="+phase2Sentinel)
	cmd.Stdin = strings.NewReader("hello\n/quit\n")
	out, err := cmd.CombinedOutput()
	if strings.Contains(string(out), phase2Sentinel) {
		t.Fatal("binary output disclosed sentinel")
	}
	if err != nil {
		t.Fatal("authenticated binary chat failed")
	}
	if !strings.Contains(string(out), "ok") {
		t.Fatal("authenticated binary missing streamed text")
	}
	if len(auths) != 2 {
		t.Fatalf("requests %d", len(auths))
	}
	if auths[0] != "" {
		t.Fatal("no-auth request sent an Authorization header")
	}
	if auths[1] != "Bearer "+phase2Sentinel {
		t.Fatal("authenticated request missing expected Bearer header")
	}
}

func TestBinaryMissingTokenNoNetwork(t *testing.T) {
	bin := buildBin(t)
	var hit atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Store(true)
	}))
	t.Cleanup(srv.Close)
	path := filepath.Join(t.TempDir(), "auth.json")
	writeAuthCfgAt(t, path, srv.URL+"/v1/chat/completions", "RADICHAT_PHASE2_TEST_UNSET")
	cmd := exec.Command(bin, "-config", path)
	cmd.Env = environWithout("RADICHAT_PHASE2_TEST_UNSET")
	cmd.Stdin = strings.NewReader("hello\n")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected failure")
	}
	if hit.Load() {
		t.Fatal("binary contacted network")
	}
	if !strings.Contains(string(out), "unset") {
		t.Fatalf("output %q", out)
	}
	if strings.Contains(string(out), phase2Sentinel) {
		t.Fatal("binary disclosed sentinel")
	}
}

func TestRunHTTPFailureEchoesOmitFromStdio(t *testing.T) {
	t.Setenv("RADICHAT_PHASE2_TEST_TOKEN", phase2Sentinel)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "echo-auth="+r.Header.Get("Authorization")+" echo-token="+phase2Sentinel+"\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	path := filepath.Join(t.TempDir(), "radichat.json")
	writeAuthCfgAt(t, path, srv.URL+"/v1/chat/completions", "RADICHAT_PHASE2_TEST_TOKEN")
	var out, errb bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", path}, strings.NewReader("ping\n/quit\n"), &out, &errb)
	if strings.Contains(out.String(), phase2Sentinel) {
		t.Fatal("stdout disclosed sentinel")
	}
	if strings.Contains(errb.String(), phase2Sentinel) {
		t.Fatal("stderr disclosed sentinel")
	}
	if code != 0 {
		t.Fatal("expected /quit after a failed turn to exit 0")
	}
	if !strings.Contains(errb.String(), "401") {
		t.Fatal("expected HTTP 401 on stderr")
	}
	if strings.Contains(errb.String(), "server said") {
		t.Fatal("stderr included untrusted response snippet")
	}
	if !strings.Contains(errb.String(), "omitted") {
		t.Fatal("expected omitted-body classification on stderr")
	}
}

func TestBinaryHTTPFailureEchoesOmitFromStdio(t *testing.T) {
	bin := buildBin(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "echo-auth="+r.Header.Get("Authorization")+" echo-token="+phase2Sentinel+"\n")
	}))
	t.Cleanup(srv.Close)
	path := filepath.Join(t.TempDir(), "auth.json")
	writeAuthCfgAt(t, path, srv.URL+"/v1/chat/completions", "RADICHAT_PHASE2_TEST_TOKEN")
	cmd := exec.Command(bin, "-config", path)
	cmd.Env = append(environWithout("RADICHAT_PHASE2_TEST_TOKEN"), "RADICHAT_PHASE2_TEST_TOKEN="+phase2Sentinel)
	cmd.Stdin = strings.NewReader("hello\n/quit\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if strings.Contains(stdout.String(), phase2Sentinel) {
		t.Fatal("binary stdout disclosed sentinel")
	}
	if strings.Contains(stderr.String(), phase2Sentinel) {
		t.Fatal("binary stderr disclosed sentinel")
	}
	if err != nil {
		t.Fatal("expected /quit after a failed turn to exit 0")
	}
	if !strings.Contains(stderr.String(), "502") {
		t.Fatal("expected HTTP 502 on stderr")
	}
	if strings.Contains(stderr.String(), "server said") {
		t.Fatal("binary stderr included untrusted response snippet")
	}
	if !strings.Contains(stderr.String(), "omitted") {
		t.Fatal("expected omitted-body classification on stderr")
	}
}

func TestBinaryAuthRedirectLocationOmitsFromStdio(t *testing.T) {
	bin := buildBin(t)
	loc := "/steal?token=" + phase2Sentinel + "&q=" + strings.ReplaceAll(phase2Sentinel, "a", "%61")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", loc)
		w.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	path := filepath.Join(t.TempDir(), "auth.json")
	writeAuthCfgAt(t, path, srv.URL+"/v1/chat/completions", "RADICHAT_PHASE2_TEST_TOKEN")
	cmd := exec.Command(bin, "-config", path)
	cmd.Env = append(environWithout("RADICHAT_PHASE2_TEST_TOKEN"), "RADICHAT_PHASE2_TEST_TOKEN="+phase2Sentinel)
	cmd.Stdin = strings.NewReader("hello\n/quit\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	if strings.Contains(stdout.String(), phase2Sentinel) {
		t.Fatal("binary stdout disclosed sentinel")
	}
	if strings.Contains(stderr.String(), phase2Sentinel) || strings.Contains(stderr.String(), "steal") {
		t.Fatal("binary stderr disclosed Location or sentinel")
	}
	if !strings.Contains(stderr.String(), "redirect") {
		t.Fatal("expected redirect classification on stderr")
	}
}

func TestBinaryAuthStreamEchoOmitsFromStdio(t *testing.T) {
	bin := buildBin(t)
	mid := len(phase2Sentinel) / 2
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"pre\"},\"finish_reason\":null}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\""+phase2Sentinel[:mid]+"\"},\"finish_reason\":null}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\""+phase2Sentinel[mid:]+"\"},\"finish_reason\":null}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"post\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	path := filepath.Join(t.TempDir(), "auth.json")
	writeAuthCfgAt(t, path, srv.URL+"/v1/chat/completions", "RADICHAT_PHASE2_TEST_TOKEN")
	cmd := exec.Command(bin, "-config", path)
	cmd.Env = append(environWithout("RADICHAT_PHASE2_TEST_TOKEN"), "RADICHAT_PHASE2_TEST_TOKEN="+phase2Sentinel)
	cmd.Stdin = strings.NewReader("hello\n/quit\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatal("authenticated stream binary exited nonzero")
	}
	if strings.Contains(stdout.String(), phase2Sentinel) || strings.Contains(stderr.String(), phase2Sentinel) {
		t.Fatal("binary disclosed sentinel in stream output")
	}
	if !strings.Contains(stdout.String(), "pre") || !strings.Contains(stdout.String(), "post") {
		t.Fatal("binary lost surrounding streamed text")
	}
}

func TestBinaryAuthTrailingTokenPrefixOmitsFromStdio(t *testing.T) {
	bin := buildBin(t)
	prefix := phase2Sentinel[:len(phase2Sentinel)-1]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"pre\"},\"finish_reason\":null}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\""+prefix+"\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	path := filepath.Join(t.TempDir(), "auth.json")
	writeAuthCfgAt(t, path, srv.URL+"/v1/chat/completions", "RADICHAT_PHASE2_TEST_TOKEN")
	cmd := exec.Command(bin, "-config", path)
	cmd.Env = append(environWithout("RADICHAT_PHASE2_TEST_TOKEN"), "RADICHAT_PHASE2_TEST_TOKEN="+phase2Sentinel)
	cmd.Stdin = strings.NewReader("hello\n/quit\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatal("authenticated trailing-prefix binary exited nonzero")
	}
	if strings.Contains(stdout.String(), prefix) || strings.Contains(stderr.String(), prefix) {
		t.Fatal("binary disclosed trailing token prefix")
	}
	if !strings.Contains(stdout.String(), "pre") {
		t.Fatal("binary lost surrounding streamed text")
	}
}

func TestBinaryAuthKeepsNonSecretTrailingOutput(t *testing.T) {
	bin := buildBin(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"},\"finish_reason\":null}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"tail-ok\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	path := filepath.Join(t.TempDir(), "auth.json")
	writeAuthCfgAt(t, path, srv.URL+"/v1/chat/completions", "RADICHAT_PHASE2_TEST_TOKEN")
	cmd := exec.Command(bin, "-config", path)
	cmd.Env = append(environWithout("RADICHAT_PHASE2_TEST_TOKEN"), "RADICHAT_PHASE2_TEST_TOKEN="+phase2Sentinel)
	cmd.Stdin = strings.NewReader("hello\n/quit\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatal("authenticated non-secret trailing binary exited nonzero")
	}
	if !strings.Contains(stdout.String(), "hellotail-ok") {
		t.Fatal("binary dropped non-secret trailing output")
	}
	if strings.Contains(stdout.String(), phase2Sentinel) || strings.Contains(stderr.String(), phase2Sentinel) {
		t.Fatal("binary disclosed sentinel")
	}
}

func TestBinaryAuthTransportErrorOmitsFromStdio(t *testing.T) {
	bin := buildBin(t)
	path := filepath.Join(t.TempDir(), "auth.json")
	writeAuthCfgAt(t, path, "http://127.0.0.1:1/v1/chat/completions", "RADICHAT_PHASE2_TEST_TOKEN")
	cmd := exec.Command(bin, "-config", path)
	cmd.Env = append(environWithout("RADICHAT_PHASE2_TEST_TOKEN"), "RADICHAT_PHASE2_TEST_TOKEN="+phase2Sentinel)
	cmd.Stdin = strings.NewReader("hello\n/quit\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	if strings.Contains(stdout.String(), phase2Sentinel) || strings.Contains(stderr.String(), phase2Sentinel) {
		t.Fatal("binary transport path disclosed sentinel")
	}
	if !strings.Contains(stderr.String(), "cannot contact") {
		t.Fatal("expected classified transport error on stderr")
	}
}

func TestBinaryZeroAuthRequestShapeUnchanged(t *testing.T) {
	bin := buildBin(t)
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("unexpected auth")
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"z\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(srv.Close)
	path := writeCfg(t, srv.URL+"/v1/chat/completions", 400, 40, "sys")
	cmd := exec.Command(bin, "-config", path)
	cmd.Stdin = strings.NewReader("hi\n/quit\n")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if body["stream"] != true || body["model"] == nil {
		t.Fatalf("body %#v", body)
	}
	if _, ok := body["tools"]; ok {
		t.Fatal("tools present")
	}
}

func environWithout(key string) []string {
	prefix := key + "="
	out := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, prefix) {
			continue
		}
		out = append(out, e)
	}
	return out
}
