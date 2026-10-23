package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestBatchOutputFileAndStdout(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"pong\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	dir := t.TempDir()
	cfgPath := filepathJoinCfg(t, dir, srv.URL+"/v1/chat/completions")
	outPath := dir + "/out.jsonl"
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", cfgPath, "-output", outPath}, strings.NewReader("hi\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, stderr.String())
	}
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	rows := decodeJSONL(t, raw)
	if len(rows) != 1 || rows[0]["output"] != "pong" || rows[0]["status"] != "ok" {
		t.Fatalf("file %s", raw)
	}
	if !strings.Contains(stdout.String(), "pong") {
		t.Fatalf("stdout should still stream, got %q", stdout.String())
	}

	var only, errb bytes.Buffer
	code = run(context.Background(), []string{"radichat", "-config", cfgPath, "-output", "-"}, strings.NewReader("hi\n"), &only, &errb)
	if code != 0 {
		t.Fatalf("stdout mode exit %d stderr %s", code, errb.String())
	}
	rows = decodeJSONL(t, only.Bytes())
	if len(rows) != 1 || rows[0]["output"] != "pong" {
		t.Fatalf("stdout jsonl %s", only.String())
	}
	if strings.Contains(errb.String(), "pong") {
		t.Fatalf("diagnostic mixed into stderr with the delta: %q", errb.String())
	}
	if hits != 2 {
		t.Fatalf("hits %d", hits)
	}
}

func TestBatchOutputExistingFileIsUntouched(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request issued")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	dir := t.TempDir()
	cfgPath := filepathJoinCfg(t, dir, srv.URL+"/v1/chat/completions")
	outPath := dir + "/out.jsonl"
	if err := os.WriteFile(outPath, []byte("KEEP\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", cfgPath, "-output", outPath}, strings.NewReader("hi\n"), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit %d stderr %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "already exists") {
		t.Fatalf("stderr %q", stderr.String())
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "KEEP\n" {
		t.Fatalf("file changed to %q", got)
	}
}

func TestBatchOutputNotCreatedWhenFitFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request issued")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	dir := t.TempDir()
	prompt := strings.Repeat("p", 95)
	raw := `{"endpoint":"` + srv.URL + `/v1/chat/completions","model":"m","context_budget":200,"generation_reserve":10,"system_prompt":"` + prompt + `"}`
	cfgPath := dir + "/radichat.json"
	if err := os.WriteFile(cfgPath, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	outPath := dir + "/out.jsonl"
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", cfgPath, "-output", outPath}, strings.NewReader("hi\n"), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit %d stderr %q", code, stderr.String())
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatalf("output created: %v", err)
	}
	if strings.Contains(stderr.String(), prompt) {
		t.Fatalf("stderr leaked prompt: %q", stderr.String())
	}
}

func TestBatchOutputDevFullStopsFurtherRequests(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	dir := t.TempDir()
	cfgPath := filepathJoinCfg(t, dir, srv.URL+"/v1/chat/completions")
	full, err := os.OpenFile("/dev/full", os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { full.Close() })
	var stderr bytes.Buffer
	code := run(context.Background(), []string{"radichat", "-config", cfgPath, "-output", "-"}, strings.NewReader("one\ntwo\n"), full, &stderr)
	if code != 1 {
		t.Fatalf("exit %d stderr %s", code, stderr.String())
	}
	if hits != 1 {
		t.Fatalf("requests after write failure: %d", hits)
	}
}

func filepathJoinCfg(t *testing.T, dir, endpoint string) string {
	t.Helper()
	path := dir + "/radichat.json"
	writeCfgAt(t, path, endpoint, 400, 40, "")
	return path
}

func decodeJSONL(t *testing.T, b []byte) []map[string]any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(b))
	var rows []map[string]any
	for {
		var row map[string]any
		if err := dec.Decode(&row); err != nil {
			if err == io.EOF {
				return rows
			}
			t.Fatalf("jsonl %v in %s", err, b)
		}
		rows = append(rows, row)
	}
}
