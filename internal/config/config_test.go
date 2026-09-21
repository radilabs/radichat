package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseValidWithSystemPrompt(t *testing.T) {
	cfg, err := Parse("t.json", []byte(`{
		"endpoint": "http://127.0.0.1:8080/v1/chat/completions",
		"model": "local-model",
		"context_budget": 8192,
		"generation_reserve": 1024,
		"system_prompt": "Be brief."
	}`))
	if err != nil {
		t.Fatalf("valid config: %v", err)
	}
	if cfg.Endpoint != "http://127.0.0.1:8080/v1/chat/completions" {
		t.Fatalf("endpoint: %q", cfg.Endpoint)
	}
	if cfg.Model != "local-model" {
		t.Fatalf("model: %q", cfg.Model)
	}
	if cfg.ContextBudget != 8192 || cfg.GenerationReserve != 1024 {
		t.Fatalf("budgets: %d %d", cfg.ContextBudget, cfg.GenerationReserve)
	}
	if cfg.SystemPrompt != "Be brief." {
		t.Fatalf("system: %q", cfg.SystemPrompt)
	}
}

func TestParseValidWithoutSystemPrompt(t *testing.T) {
	cfg, err := Parse("t.json", []byte(`{
		"endpoint": "https://example.invalid/v1/chat/completions",
		"model": "m",
		"context_budget": 100,
		"generation_reserve": 10
	}`))
	if err != nil {
		t.Fatalf("valid config: %v", err)
	}
	if cfg.SystemPrompt != "" {
		t.Fatalf("expected empty system prompt, got %q", cfg.SystemPrompt)
	}
}

func TestParseTrailingWhitespaceOK(t *testing.T) {
	_, err := Parse("t.json", []byte(`{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10}
`))
	if err != nil {
		t.Fatalf("trailing whitespace should be allowed: %v", err)
	}
}

func TestParseRejects(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "   ", want: "empty"},
		{name: "malformed", raw: "{", want: "malformed JSON"},
		{name: "array", raw: `[]`, want: "JSON object"},
		{name: "unknown key", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"api_key":"secret"}`, want: "unknown field"},
		{name: "auth field", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"authorization":"Bearer x"}`, want: "unknown field"},
		{name: "duplicate key", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","model":"n","context_budget":100,"generation_reserve":10}`, want: "duplicate JSON key"},
		{name: "trailing value", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10}{}`, want: "trailing JSON"},
		{name: "missing endpoint", raw: `{"model":"m","context_budget":100,"generation_reserve":10}`, want: "missing endpoint"},
		{name: "missing model", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","context_budget":100,"generation_reserve":10}`, want: "missing model"},
		{name: "missing budget", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","generation_reserve":10}`, want: "missing context_budget"},
		{name: "missing reserve", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100}`, want: "missing generation_reserve"},
		{name: "empty model", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"  ","context_budget":100,"generation_reserve":10}`, want: "nonempty string"},
		{name: "empty endpoint", raw: `{"endpoint":"","model":"m","context_budget":100,"generation_reserve":10}`, want: "nonempty string"},
		{name: "string budget", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":"8192","generation_reserve":10}`, want: "positive integer"},
		{name: "float budget", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":8192.5,"generation_reserve":10}`, want: "positive decimal integer"},
		{name: "bool model", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":true,"context_budget":100,"generation_reserve":10}`, want: "JSON string"},
		{name: "zero budget", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":0,"generation_reserve":10}`, want: "positive integer"},
		{name: "negative reserve", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":-1}`, want: "positive integer"},
		{name: "reserve equal budget", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":100}`, want: "smaller than context_budget"},
		{name: "reserve larger", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":101}`, want: "smaller than context_budget"},
		{name: "url query", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions?x=1","model":"m","context_budget":100,"generation_reserve":10}`, want: "query string"},
		{name: "url fragment", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions#frag","model":"m","context_budget":100,"generation_reserve":10}`, want: "fragment"},
		{name: "url userinfo", raw: `{"endpoint":"http://user:pass@127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10}`, want: "userinfo"},
		{name: "url ftp", raw: `{"endpoint":"ftp://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10}`, want: "http:// or https://"},
		{name: "relative url", raw: `{"endpoint":"/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10}`, want: "absolute"},
		{name: "system prompt number", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"system_prompt":1}`, want: "JSON string"},
		{name: "system prompt null", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"system_prompt":null}`, want: "omit the field instead of using null"},
		{name: "system prompt bool", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"system_prompt":true}`, want: "JSON string"},
		{name: "system prompt array", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"system_prompt":[]}`, want: "JSON string"},
		{name: "leading zeros", raw: `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":0100,"generation_reserve":10}`, want: "malformed JSON"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("t.json", []byte(tc.raw))
			if err == nil {
				t.Fatalf("expected error containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.want)
			}
			if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "Bearer x") {
				t.Fatalf("error dumped a secret: %q", err.Error())
			}
		})
	}
}

func TestLoadMissingAndDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(filepath.Join(dir, "missing.json"))
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing file: %v", err)
	}
	_, err = Load(dir)
	if err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("directory: %v", err)
	}
}

func TestLoadValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "radichat.json")
	body := `{
		"endpoint": "http://127.0.0.1:9/v1/chat/completions",
		"model": "m",
		"context_budget": 200,
		"generation_reserve": 20
	}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "m" || cfg.ContextBudget != 200 {
		t.Fatalf("loaded %+v", cfg)
	}
}

func TestParseDoesNotAppendPath(t *testing.T) {
	cfg, err := Parse("t.json", []byte(`{
		"endpoint": "http://127.0.0.1:8080/custom",
		"model": "m",
		"context_budget": 100,
		"generation_reserve": 10
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Endpoint != "http://127.0.0.1:8080/custom" {
		t.Fatalf("endpoint rewritten: %q", cfg.Endpoint)
	}
}

func TestParseEmptySystemPromptString(t *testing.T) {
	cfg, err := Parse("t.json", []byte(`{
		"endpoint": "http://127.0.0.1/v1/chat/completions",
		"model": "m",
		"context_budget": 100,
		"generation_reserve": 10,
		"system_prompt": ""
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SystemPrompt != "" {
		t.Fatalf("empty string should be kept as empty, got %q", cfg.SystemPrompt)
	}
}

func TestEncodingJSONNullIntoStringDoesNotError(t *testing.T) {
	// encoding/json does not treat JSON null as a type error for string.
	// Phase 1 still rejects system_prompt:null because the approved field is
	// an optional string: omit the key, or supply a JSON string.
	s := "unchanged"
	if err := json.Unmarshal([]byte("null"), &s); err != nil {
		t.Fatalf("stdlib json.Unmarshal(null, *string) error: %v", err)
	}
}

func TestParseAllowsIPv6(t *testing.T) {
	_, err := Parse("t.json", []byte(`{
		"endpoint": "http://[::1]:8080/v1/chat/completions",
		"model": "m",
		"context_budget": 100,
		"generation_reserve": 10
	}`))
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoadExampleFile(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "radichat.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model == "" || cfg.Endpoint == "" || cfg.GenerationReserve >= cfg.ContextBudget {
		t.Fatalf("example config invalid: %+v", cfg)
	}
}
