package config

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func promptConfig(extra string) string {
	return `{
		"endpoint": "http://127.0.0.1:9/v1/chat/completions",
		"model": "m",
		"context_budget": 200,
		"generation_reserve": 10,
		` + extra + `
	}`
}

func TestSystemPromptFileParseStoresPathWithoutReading(t *testing.T) {
	raw := promptConfig(`"system_prompt_file": "missing-prompt.txt"`)
	cfg, err := Parse("cfg/radichat.json", []byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.SystemPromptFile != "missing-prompt.txt" {
		t.Fatalf("path %q", cfg.SystemPromptFile)
	}
	if cfg.SystemPrompt != "" {
		t.Fatalf("Parse read prompt text %q", cfg.SystemPrompt)
	}
}

func TestSystemPromptFileParseRejects(t *testing.T) {
	secret := "prompt-bytes-should-not-leak"
	cases := []struct {
		name  string
		extra string
		want  string
	}{
		{
			name:  "conflicts with inline",
			extra: `"system_prompt": "hello", "system_prompt_file": "p.txt"`,
			want:  "conflict",
		},
		{
			name:  "conflicts with empty inline",
			extra: `"system_prompt": "", "system_prompt_file": "p.txt"`,
			want:  "conflict",
		},
		{
			name:  "null",
			extra: `"system_prompt_file": null`,
			want:  "null",
		},
		{
			name:  "number",
			extra: `"system_prompt_file": 1`,
			want:  "JSON string",
		},
		{
			name:  "bool",
			extra: `"system_prompt_file": true`,
			want:  "JSON string",
		},
		{
			name:  "empty",
			extra: `"system_prompt_file": ""`,
			want:  "nonempty",
		},
		{
			name:  "whitespace",
			extra: `"system_prompt_file": "  "`,
			want:  "nonempty",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("t.json", []byte(promptConfig(tc.extra)))
			if err == nil {
				t.Fatalf("expected error containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.want)
			}
			if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "hello") {
				t.Fatalf("error included prompt material: %q", err.Error())
			}
		})
	}
}

func TestSystemPromptFileLoad(t *testing.T) {
	t.Run("relative to config directory", func(t *testing.T) {
		dir := t.TempDir()
		cfgDir := filepath.Join(dir, "cfg")
		promptDir := filepath.Join(cfgDir, "prompts")
		if err := os.MkdirAll(promptDir, 0o700); err != nil {
			t.Fatal(err)
		}
		promptPath := filepath.Join(promptDir, "system.txt")
		if err := os.WriteFile(promptPath, []byte("from-file"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(cfgDir, "radichat.json")
		if err := os.WriteFile(cfgPath, []byte(promptConfig(`"system_prompt_file": "prompts/system.txt"`)), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.SystemPrompt != "from-file" {
			t.Fatalf("prompt %q", cfg.SystemPrompt)
		}
	})

	t.Run("absolute path", func(t *testing.T) {
		dir := t.TempDir()
		promptPath := filepath.Join(dir, "abs.txt")
		if err := os.WriteFile(promptPath, []byte("absolute-prompt"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfgDir := filepath.Join(dir, "elsewhere")
		if err := os.Mkdir(cfgDir, 0o700); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(cfgDir, "radichat.json")
		raw := promptConfig(`"system_prompt_file": ` + jsonQuote(promptPath))
		if err := os.WriteFile(cfgPath, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.SystemPrompt != "absolute-prompt" {
			t.Fatalf("prompt %q", cfg.SystemPrompt)
		}
	})

	t.Run("symlink to regular file", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target.txt")
		if err := os.WriteFile(target, []byte("via-symlink"), 0o600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "link.txt")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(dir, "radichat.json")
		raw := promptConfig(`"system_prompt_file": "link.txt"`)
		if err := os.WriteFile(cfgPath, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.SystemPrompt != "via-symlink" {
			t.Fatalf("prompt %q", cfg.SystemPrompt)
		}
	})
}

func TestSystemPromptFileLoadRejects(t *testing.T) {
	secret := "query-prompt-super-secret-value"

	t.Run("directory", func(t *testing.T) {
		dir := t.TempDir()
		promptDir := filepath.Join(dir, "not-a-file")
		if err := os.Mkdir(promptDir, 0o700); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(dir, "radichat.json")
		if err := os.WriteFile(cfgPath, []byte(promptConfig(`"system_prompt_file": "not-a-file"`)), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Load(cfgPath)
		if err == nil || !strings.Contains(err.Error(), "directory") {
			t.Fatalf("directory: %v", err)
		}
	})

	t.Run("fifo", func(t *testing.T) {
		dir := t.TempDir()
		fifo := filepath.Join(dir, "prompt.fifo")
		if err := syscall.Mkfifo(fifo, 0o600); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(dir, "radichat.json")
		if err := os.WriteFile(cfgPath, []byte(promptConfig(`"system_prompt_file": "prompt.fifo"`)), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Load(cfgPath)
		if err == nil || !strings.Contains(err.Error(), "regular file") {
			t.Fatalf("fifo: %v", err)
		}
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked prompt: %q", err.Error())
		}
	})

	t.Run("oversize", func(t *testing.T) {
		dir := t.TempDir()
		promptPath := filepath.Join(dir, "big.txt")
		body := []byte(strings.Repeat("a", MaxFileBytes+1))
		if err := os.WriteFile(promptPath, body, 0o600); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(dir, "radichat.json")
		if err := os.WriteFile(cfgPath, []byte(promptConfig(`"system_prompt_file": "big.txt"`)), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Load(cfgPath)
		if err == nil || !strings.Contains(err.Error(), "limit") {
			t.Fatalf("oversize: %v", err)
		}
	})

	t.Run("invalid utf-8 omits prompt bytes", func(t *testing.T) {
		dir := t.TempDir()
		promptPath := filepath.Join(dir, "bad.txt")
		body := append([]byte(secret), 0xff)
		if err := os.WriteFile(promptPath, body, 0o600); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(dir, "radichat.json")
		if err := os.WriteFile(cfgPath, []byte(promptConfig(`"system_prompt_file": "bad.txt"`)), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Load(cfgPath)
		if err == nil || !strings.Contains(err.Error(), "UTF-8") {
			t.Fatalf("utf-8: %v", err)
		}
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked prompt: %q", err.Error())
		}
	})

	t.Run("empty after newline", func(t *testing.T) {
		dir := t.TempDir()
		for _, body := range []string{"", "\n", "\r\n", "\uFEFF", "\uFEFF\n"} {
			promptPath := filepath.Join(dir, "empty.txt")
			if err := os.WriteFile(promptPath, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			cfgPath := filepath.Join(dir, "radichat.json")
			if err := os.WriteFile(cfgPath, []byte(promptConfig(`"system_prompt_file": "empty.txt"`)), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Load(cfgPath)
			if err == nil || !strings.Contains(err.Error(), "empty") {
				t.Fatalf("body %q: %v", body, err)
			}
		}
	})

	t.Run("missing", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "radichat.json")
		if err := os.WriteFile(cfgPath, []byte(promptConfig(`"system_prompt_file": "nope.txt"`)), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := Load(cfgPath)
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("missing: %v", err)
		}
	})
}

func TestSystemPromptFileStripsBOMAndOneTerminator(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "bom", body: "\uFEFFhello", want: "hello"},
		{name: "lf", body: "hello\n", want: "hello"},
		{name: "crlf", body: "hello\r\n", want: "hello"},
		{name: "one of two lf", body: "hello\n\n", want: "hello\n"},
		{name: "one of two crlf", body: "hello\r\n\r\n", want: "hello\r\n"},
		{name: "bom and crlf", body: "\uFEFFhello\r\n", want: "hello"},
		{name: "interior newline kept", body: "hello\nworld\n", want: "hello\nworld"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "p.txt"), []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			cfgPath := filepath.Join(dir, "radichat.json")
			if err := os.WriteFile(cfgPath, []byte(promptConfig(`"system_prompt_file": "p.txt"`)), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(cfgPath)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.SystemPrompt != tc.want {
				t.Fatalf("got %q want %q", cfg.SystemPrompt, tc.want)
			}
		})
	}
}

func TestSystemPromptFileExactSizeAccepted(t *testing.T) {
	dir := t.TempDir()
	body := strings.Repeat("b", MaxFileBytes)
	if err := os.WriteFile(filepath.Join(dir, "p.txt"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "radichat.json")
	raw := `{
		"endpoint": "http://127.0.0.1:9/v1/chat/completions",
		"model": "m",
		"context_budget": 2000000,
		"generation_reserve": 10,
		"system_prompt_file": "p.txt"
	}`
	if err := os.WriteFile(cfgPath, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SystemPrompt != body {
		t.Fatalf("len %d", len(cfg.SystemPrompt))
	}
}

func jsonQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `\`, `\\`) + `"`
}
