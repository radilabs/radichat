package mockllm

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// startServer starts a mock server and registers cleanup.
func startServer(t *testing.T, opts Options) *Server {
	t.Helper()
	srv, err := Start(opts)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		if err := srv.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return srv
}

func TestBindsLoopbackOnlyByDefault(t *testing.T) {
	srv := startServer(t, Options{})
	host, _, err := net.SplitHostPort(srv.Addr())
	if err != nil {
		t.Fatalf("Addr %q: %v", srv.Addr(), err)
	}
	if !net.ParseIP(host).IsLoopback() {
		t.Fatalf("bound to non-loopback host %q", host)
	}
	if !strings.HasPrefix(srv.URL(), "http://127.0.0.1:") {
		t.Fatalf("unexpected URL %q", srv.URL())
	}
}

func TestRefusesNonLoopbackBind(t *testing.T) {
	for _, addr := range []string{"0.0.0.0:0", "192.168.1.10:0", "[2001:db8::1]:0"} {
		if _, err := New(Options{Addr: addr}); err == nil {
			t.Errorf("New(%q) succeeded; want loopback refusal", addr)
		}
	}
}

func TestAcceptsLoopbackVariants(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:0", "127.0.0.5:0", "[::1]:0", "localhost:0"} {
		if _, err := New(Options{Addr: addr}); err != nil {
			t.Errorf("New(%q) failed: %v", addr, err)
		}
	}
}

func TestCloseIsIdempotentAndStopsServing(t *testing.T) {
	srv := startServer(t, Options{})
	if err := srv.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := srv.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	select {
	case <-srv.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after Close")
	}
	if err := srv.ServeErr(); err != nil {
		t.Fatalf("ServeErr: %v", err)
	}
}

func TestStartContextStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	srv, err := StartContext(ctx, Options{})
	if err != nil {
		t.Fatalf("StartContext: %v", err)
	}
	cancel()
	select {
	case <-srv.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after context cancel")
	}
}

func TestShutdownDrainsCleanly(t *testing.T) {
	srv := startServer(t, Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	select {
	case <-srv.Done():
	default:
		t.Fatal("Done not closed after Shutdown")
	}
}

func TestUnknownScenarioIsClear(t *testing.T) {
	srv := startServer(t, Options{})
	resp, err := httpPost(srv.URL() + "/v1/chat/completions/does-not-exist")
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("status %d; want 404", resp.StatusCode)
	}
	rec, ok := srv.Last()
	if !ok {
		t.Fatal("request was not recorded")
	}
	if rec.Scenario != "" {
		t.Fatalf("unknown request recorded a scenario %q", rec.Scenario)
	}
}
