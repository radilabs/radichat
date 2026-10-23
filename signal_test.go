package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestSignalSIGTERMDuringStream(t *testing.T) {
	bin := buildBin(t)
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"SECRETPARTIAL\"},\"finish_reason\":null}]}\n\n")
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
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "radichat.json")
	writeCfgAt(t, cfgPath, srv.URL+"/v1/chat/completions", 400, 40, "")
	outPath := filepath.Join(dir, "out.jsonl")
	cmd := exec.Command(bin, "-config", cfgPath, "-output", outPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
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
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		code := exitCode(t, err)
		if code != 143 {
			t.Fatalf("exit %d stderr %s", code, stderr.String())
		}
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("hang after SIGTERM")
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "SECRETPARTIAL") || strings.Contains(string(raw), `"status":"ok"`) {
		t.Fatalf("partial or success record: %s", raw)
	}
	if !strings.Contains(string(raw), `"status":"interrupted"`) {
		t.Fatalf("missing interrupted record: %s", raw)
	}
}

func TestSignalSIGTERMWhileIdle(t *testing.T) {
	bin := buildBin(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "radichat.json")
	writeCfgAt(t, cfgPath, "http://127.0.0.1:9/v1/chat/completions", 400, 40, "")
	cmd := exec.Command(bin, "-config", cfgPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	time.Sleep(200 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		code := exitCode(t, err)
		if code != 143 {
			t.Fatalf("exit %d stderr %s", code, stderr.String())
		}
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("hang after idle SIGTERM")
	}
	if !strings.Contains(stderr.String(), "interrupted") {
		t.Fatalf("stderr %q", stderr.String())
	}
}

func TestSignalSIGINTStillExits130(t *testing.T) {
	bin := buildBin(t)
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"},\"finish_reason\":null}]}\n\n")
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
		if code := exitCode(t, err); code != 130 {
			t.Fatalf("exit %d", code)
		}
	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("hang after SIGINT")
	}
}

func exitCode(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	ee, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatal(err)
	}
	return ee.ExitCode()
}
