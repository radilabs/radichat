package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/radilabs/radichat/internal/mockllm"
)

// sentinel is an obviously fake credential used only by these tests.
const sentinel = "mockllm-sentinel-token-not-a-real-credential"

// TestListPrintsEveryScenario verifies the list command works and covers the
// catalog.
func TestListPrintsEveryScenario(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"radichat-mock", "list"}, &out, &errb)
	if code != 0 {
		t.Fatalf("exit %d; stderr %s", code, errb.String())
	}
	got := out.String()
	for _, want := range []string{"stream-success-basic", "auth-required-match", "fragment-inside-done", "context-system-retained"} {
		if !strings.Contains(got, want) {
			t.Errorf("listing missing %s", want)
		}
	}
	for _, family := range []string{"stream", "framing", "malformed", "http-error", "auth", "redirect", "context"} {
		if !strings.Contains(got, family) {
			t.Errorf("listing missing family %s", family)
		}
	}
}

func TestHelpExitsZero(t *testing.T) {
	for _, arg := range []string{"-help", "-h", "--help", "help"} {
		var out, errb bytes.Buffer
		code := run([]string{"radichat-mock", arg}, &out, &errb)
		if code != 0 {
			t.Errorf("%s: exit %d", arg, code)
		}
		if !strings.Contains(out.String(), "Usage:") {
			t.Errorf("%s: no usage text", arg)
		}
	}
}

func TestNoArgsExitsTwo(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"radichat-mock"}, &out, &errb); code != 2 {
		t.Fatalf("exit %d; want 2", code)
	}
}

func TestUnknownCommandExitsTwo(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"radichat-mock", "frobnicate"}, &out, &errb); code != 2 {
		t.Fatalf("exit %d; want 2", code)
	}
}

func TestUnknownScenarioExitsTwo(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"radichat-mock", "run", "nope"}, &out, &errb); code != 2 {
		t.Fatalf("exit %d; want 2", code)
	}
	if !strings.Contains(errb.String(), "unknown scenario") {
		t.Errorf("stderr %q", errb.String())
	}
}

func TestRefusesNonLoopbackAddr(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"radichat-mock", "run", "stream-success-basic", "-addr", "0.0.0.0:0"}, &out, &errb); code != 2 {
		t.Fatalf("exit %d; want 2", code)
	}
	if !strings.Contains(errb.String(), "non-loopback") {
		t.Errorf("stderr %q", errb.String())
	}
}

// TestFlagsAcceptedAfterScenarioName guards the reordering: Go's flag parser
// stops at the first non-flag argument, so a trailing -addr must still be
// parsed instead of silently starting a default listener.
//
// The valid-address case is exercised end to end by
// TestRunServesAndShutsDownCleanly, which runs the CLI in a child process.
func TestFlagsAcceptedAfterScenarioName(t *testing.T) {
	flags, positional := splitFlags([]string{"stream-success-basic", "-addr", "0.0.0.0:0"})
	if len(positional) != 1 || positional[0] != "stream-success-basic" {
		t.Fatalf("positional %v", positional)
	}
	if len(flags) != 2 || flags[0] != "-addr" || flags[1] != "0.0.0.0:0" {
		t.Fatalf("flags %v", flags)
	}
	// Same order must still be refused rather than serving.
	var out, errb bytes.Buffer
	if code := run([]string{"radichat-mock", "run", "stream-success-basic", "-addr", "0.0.0.0:0"}, &out, &errb); code != 2 {
		t.Fatalf("exit %d; want 2", code)
	}
}

func TestSplitFlagsVariants(t *testing.T) {
	flags, positional := splitFlags([]string{"-addr=127.0.0.1:0", "scenario-a", "--", "-not-a-flag"})
	if len(positional) != 2 || positional[0] != "scenario-a" || positional[1] != "-not-a-flag" {
		t.Fatalf("positional %v", positional)
	}
	if len(flags) != 1 || flags[0] != "-addr=127.0.0.1:0" {
		t.Fatalf("flags %v", flags)
	}
	// A lone "-" is positional, not a flag.
	_, positional = splitFlags([]string{"-"})
	if len(positional) != 1 || positional[0] != "-" {
		t.Fatalf("lone dash positional %v", positional)
	}
}

// TestRunServesAndShutsDownCleanly drives the real CLI process: it starts a
// scenario, reads the printed endpoint, exercises it, then interrupts.
func TestRunServesAndShutsDownCleanly(t *testing.T) {
	bin := buildMock(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "run", "stream-success-basic")
	var stdout syncBuffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	url := waitForEndpoint(t, &stdout, 10*time.Second)
	if !strings.HasPrefix(url, "http://127.0.0.1:") {
		t.Fatalf("endpoint is not loopback: %q", url)
	}
	if !strings.HasSuffix(url, "/v1/chat/completions/stream-success-basic") {
		t.Fatalf("unexpected endpoint %q", url)
	}

	resp, err := httpPostJSON(url, "", `{"model":"m","messages":[{"role":"user","content":"hi"}],"stream":true,"max_tokens":64}`)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(body), `"content":"hello"`) {
		t.Fatalf("body %q", string(body))
	}

	// Interrupt and confirm a clean exit.
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("signal: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("clean exit expected, got %v; stderr %s", err, stderr.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("CLI did not exit after SIGINT")
	}
	if !strings.Contains(stdout.String(), "shutting down") {
		t.Errorf("stdout missing shutdown notice: %s", stdout.String())
	}
}

// TestRootBuildExcludesMock verifies the RadiChat root build produces only the
// RadiChat binary and does not behave like the mock CLI.
func TestRootBuildExcludesMock(t *testing.T) {
	bin := buildRoot(t)
	// The root binary must not accept the mock's list command.
	var out, errb bytes.Buffer
	cmd := exec.Command(bin, "list")
	cmd.Stdout = &out
	cmd.Stderr = &errb
	_ = cmd.Run()
	if strings.Contains(out.String(), "scenarios:") {
		t.Fatal("root binary behaves like the mock CLI")
	}
	// And it must still work as RadiChat.
	cmd = exec.Command(bin, "-help")
	out.Reset()
	errb.Reset()
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		t.Fatalf("radichat -help: %v; stderr %s", err, errb.String())
	}
	if !strings.Contains(out.String(), "Usage: radichat") {
		t.Fatalf("unexpected help output %q", out.String())
	}
}

// TestMockBinaryServesScenarioForRadiChat is the known-supported smoke path:
// the real RadiChat binary completes a streamed chat against the mock.
func TestMockBinaryServesScenarioForRadiChat(t *testing.T) {
	mockBin := buildMock(t)
	radBin := buildRoot(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockCmd := exec.CommandContext(ctx, mockBin, "run", "stream-success-basic")
	var mockOut syncBuffer
	var mockErr bytes.Buffer
	mockCmd.Stdout = &mockOut
	mockCmd.Stderr = &mockErr
	if err := mockCmd.Start(); err != nil {
		t.Fatalf("start mock: %v", err)
	}
	defer func() {
		_ = mockCmd.Process.Kill()
		_, _ = mockCmd.Process.Wait()
	}()
	url := waitForEndpoint(t, &mockOut, 10*time.Second)

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "radichat.json")
	cfg := `{"endpoint":"` + url + `","model":"m","context_budget":4096,"generation_reserve":512,"system_prompt":"sys"}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	chatCtx, chatCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer chatCancel()
	chatCmd := exec.CommandContext(chatCtx, radBin, "-config", cfgPath)
	chatCmd.Stdin = strings.NewReader("hello\n/quit\n")
	var chatOut, chatErr bytes.Buffer
	chatCmd.Stdout = &chatOut
	chatCmd.Stderr = &chatErr
	if err := chatCmd.Run(); err != nil {
		t.Fatalf("radichat chat: %v; stderr %s", err, chatErr.String())
	}
	if !strings.Contains(chatOut.String(), "hello") {
		t.Fatalf("expected streamed assistant text, got stdout %q stderr %q", chatOut.String(), chatErr.String())
	}
}

// TestAuthenticatedSmokePathWithSentinel verifies the mock's auth scenario
// accepts the sentinel credential and that the sentinel never reaches stdout
// or stderr.
func TestAuthenticatedSmokePathWithSentinel(t *testing.T) {
	mockBin := buildMock(t)
	radBin := buildRoot(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mockCmd := exec.CommandContext(ctx, mockBin, "run", "auth-required-match")
	var mockOut syncBuffer
	var mockErr syncBuffer
	mockCmd.Stdout = &mockOut
	mockCmd.Stderr = &mockErr
	if err := mockCmd.Start(); err != nil {
		t.Fatalf("start mock: %v", err)
	}
	defer func() {
		_ = mockCmd.Process.Kill()
		_, _ = mockCmd.Process.Wait()
	}()
	url := waitForEndpoint(t, &mockOut, 10*time.Second)

	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "radichat.json")
	cfg := `{"endpoint":"` + url + `","model":"m","context_budget":4096,"generation_reserve":512,"bearer_token_env":"MOCKLLM_TEST_SENTINEL"}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}

	chatCtx, chatCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer chatCancel()
	chatCmd := exec.CommandContext(chatCtx, radBin, "-config", cfgPath)
	chatCmd.Env = append(os.Environ(), "MOCKLLM_TEST_SENTINEL="+sentinel)
	chatCmd.Stdin = strings.NewReader("hello\n/quit\n")
	var chatOut, chatErr bytes.Buffer
	chatCmd.Stdout = &chatOut
	chatCmd.Stderr = &chatErr
	if err := chatCmd.Run(); err != nil {
		t.Fatalf("authenticated chat: %v; stderr %s", err, chatErr.String())
	}
	if !strings.Contains(chatOut.String(), "authorized") {
		t.Fatalf("expected streamed assistant text, got stdout %q", chatOut.String())
	}
	// The sentinel credential must not appear in any command output.
	if strings.Contains(chatOut.String(), sentinel) {
		t.Error("stdout disclosed the sentinel credential")
	}
	if strings.Contains(chatErr.String(), sentinel) {
		t.Error("stderr disclosed the sentinel credential")
	}
	if strings.Contains(mockOut.String(), sentinel) || strings.Contains(mockErr.String(), sentinel) {
		t.Error("mock CLI output disclosed the sentinel credential")
	}
}

// TestRecordedRequestHasNoCredential drives the mock over HTTP and asserts the
// recorded observation contains no credential value.
func TestRecordedRequestHasNoCredential(t *testing.T) {
	srv, err := mockllm.Start(mockllm.Options{ExpectedBearer: sentinel})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })

	resp, err := httpPostJSON(srv.URL()+"/v1/chat/completions/auth-required-match",
		"Bearer "+sentinel, `{"model":"m","messages":[],"stream":true}`)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	rec, ok := srv.Last()
	if !ok {
		t.Fatal("request not recorded")
	}
	blob, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), sentinel) {
		t.Fatalf("recorded observation contains the credential: %s", blob)
	}
	// json.Marshal escapes "<" and ">", so compare against the escaped form.
	want := strings.NewReplacer("<", "\\u003c", ">", "\\u003e").Replace(mockllm.RedactedAuthorization)
	if !strings.Contains(string(blob), want) {
		t.Fatalf("recorded observation lacks the redaction placeholder: %s", blob)
	}
}

// syncBuffer is a mutex-protected buffer. exec.Cmd copies process output into
// it from its own goroutine, so reading it from the test goroutine requires
// synchronization to be race-free.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// waitForEndpoint polls a CLI's stdout for the printed endpoint line.
func waitForEndpoint(t *testing.T, stdout *syncBuffer, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, line := range strings.Split(stdout.String(), "\n") {
			if v, ok := strings.CutPrefix(line, "endpoint:"); ok {
				return strings.TrimSpace(v)
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no endpoint printed within %v; stdout %q", timeout, stdout.String())
	return ""
}

// noRedirectClient never follows redirects.
var noRedirectClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func httpPostJSON(url, auth, body string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	return noRedirectClient.Do(req)
}

func buildMock(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "radichat-mock")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build mock: %v; %s", err, out)
	}
	return bin
}

func buildRoot(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "radichat-root")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	// The root build must run from the repository root, not this package's
	// directory, so it produces the RadiChat binary.
	cmd.Dir = repoRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build root: %v; %s", err, out)
	}
	return bin
}

// repoRoot returns the absolute path of the repository root.
//
// Go runs tests with the working directory set to the package directory
// (cmd/radichat-mock), so the root is two levels up.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Dir(filepath.Dir(wd))
}
