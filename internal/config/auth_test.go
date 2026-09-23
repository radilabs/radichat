package config

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSentinel = "phase2-test-sentinel-aa11"

func validAuthJSON(envName string) string {
	return `{
		"endpoint": "http://127.0.0.1:8080/v1/chat/completions",
		"model": "m",
		"context_budget": 100,
		"generation_reserve": 10,
		"bearer_token_env": "` + envName + `"
	}`
}

func TestParseOmitsBearerKeepsZeroAuth(t *testing.T) {
	cfg, err := Parse("t.json", []byte(`{
		"endpoint": "http://127.0.0.1:8080/v1/chat/completions",
		"model": "m",
		"context_budget": 100,
		"generation_reserve": 10
	}`))
	if err != nil {
		t.Fatalf("valid config: %v", err)
	}
	if cfg.BearerTokenEnv != "" || cfg.BearerToken != "" {
		t.Fatalf("zero-auth path must not resolve a token")
	}
}

func TestParseResolvesBearerFromEnv(t *testing.T) {
	t.Setenv("RADICHAT_PHASE2_TEST_TOKEN", testSentinel)
	cfg, err := Parse("t.json", []byte(validAuthJSON("RADICHAT_PHASE2_TEST_TOKEN")))
	if err != nil {
		t.Fatalf("valid auth config: %v", err)
	}
	if cfg.BearerTokenEnv != "RADICHAT_PHASE2_TEST_TOKEN" {
		t.Fatalf("env name %q", cfg.BearerTokenEnv)
	}
	if cfg.BearerToken != testSentinel {
		t.Fatal("resolved token did not match the process environment sentinel")
	}
}

func TestParseBearerRejectsWithoutNetworkOrDisclosure(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	accepted := make(chan struct{}, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		select {
		case accepted <- struct{}{}:
		default:
		}
	}()

	cases := []struct {
		name   string
		raw    string
		setup  func(*testing.T)
		want   string
		secret string
	}{
		{
			name: "null field",
			raw:  `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"bearer_token_env":null}`,
			want: "omit the field instead of using null",
		},
		{
			name: "empty name",
			raw:  `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"bearer_token_env":""}`,
			want: "nonempty",
		},
		{
			name: "whitespace name",
			raw:  `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"bearer_token_env":"  "}`,
			want: "nonempty",
		},
		{
			name: "invalid name space",
			raw:  `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"bearer_token_env":"FOO BAR"}`,
			want: "environment variable name",
		},
		{
			name: "invalid name equals",
			raw:  `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"bearer_token_env":"FOO=BAR"}`,
			want: "environment variable name",
		},
		{
			name: "invalid name leading digit",
			raw:  `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"bearer_token_env":"1TOKEN"}`,
			want: "environment variable name",
		},
		{
			name: "invalid name hyphen",
			raw:  `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"bearer_token_env":"FOO-BAR"}`,
			want: "environment variable name",
		},
		{
			name: "number type",
			raw:  `{"endpoint":"http://127.0.0.1/v1/chat/completions","model":"m","context_budget":100,"generation_reserve":10,"bearer_token_env":1}`,
			want: "JSON string",
		},
		{
			name: "unset env",
			raw:  validAuthJSON("RADICHAT_PHASE2_TEST_UNSET"),
			setup: func(t *testing.T) {
				t.Helper()
				_ = os.Unsetenv("RADICHAT_PHASE2_TEST_UNSET")
			},
			want: "unset",
		},
		{
			name: "empty env",
			raw:  validAuthJSON("RADICHAT_PHASE2_TEST_EMPTY"),
			setup: func(t *testing.T) {
				t.Setenv("RADICHAT_PHASE2_TEST_EMPTY", "")
			},
			want: "empty",
		},
		{
			name:   "token with lf",
			raw:    validAuthJSON("RADICHAT_PHASE2_TEST_LF"),
			setup:  func(t *testing.T) { t.Setenv("RADICHAT_PHASE2_TEST_LF", "good"+"\n"+"token") },
			want:   "HTTP header",
			secret: "good\ntoken",
		},
		{
			name:   "token with cr",
			raw:    validAuthJSON("RADICHAT_PHASE2_TEST_CR"),
			setup:  func(t *testing.T) { t.Setenv("RADICHAT_PHASE2_TEST_CR", "good\rtoken") },
			want:   "HTTP header",
			secret: "good\rtoken",
		},
		{
			name:   "token with space",
			raw:    validAuthJSON("RADICHAT_PHASE2_TEST_SP"),
			setup:  func(t *testing.T) { t.Setenv("RADICHAT_PHASE2_TEST_SP", "not a header") },
			want:   "HTTP header",
			secret: "not a header",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}
			_, err := Parse("t.json", []byte(tc.raw))
			if err == nil {
				t.Fatalf("expected error containing %q", tc.want)
			}
			msg := err.Error()
			if strings.Contains(msg, testSentinel) {
				t.Fatal("error disclosed the test sentinel")
			}
			if tc.secret != "" && strings.Contains(msg, tc.secret) {
				t.Fatal("error disclosed the rejected token value")
			}
			if !strings.Contains(msg, tc.want) {
				t.Fatalf("error does not contain %q", tc.want)
			}
			select {
			case <-accepted:
				t.Fatal("validation contacted the network")
			default:
			}
		})
	}
}

func TestLoadResolvesBearerAndExampleStaysZeroAuth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "radichat.json")
	t.Setenv("RADICHAT_PHASE2_TEST_TOKEN", testSentinel)
	body := validAuthJSON("RADICHAT_PHASE2_TEST_TOKEN")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BearerToken != testSentinel {
		t.Fatal("Load did not resolve the sentinel from the environment")
	}

	example, err := Load(filepath.Join("..", "..", "radichat.example.json"))
	if err != nil {
		t.Fatal(err)
	}
	if example.BearerTokenEnv != "" || example.BearerToken != "" {
		t.Fatal("public example must remain zero-auth")
	}
}
