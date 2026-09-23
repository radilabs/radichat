package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultPath   = "radichat.json"
	MaxFileBytes  = 1 << 20
	allowedFields = "endpoint, model, context_budget, generation_reserve, system_prompt, bearer_token_env"
	maxEnvName    = 256
	maxTokenBytes = 8 << 10
)

// Config is the validated runtime configuration.
type Config struct {
	Endpoint          string
	Model             string
	ContextBudget     int64
	GenerationReserve int64
	SystemPrompt      string
	Path              string
	BearerTokenEnv    string
	BearerToken       string
}

type fileConfig struct {
	Endpoint          json.RawMessage `json:"endpoint"`
	Model             json.RawMessage `json:"model"`
	ContextBudget     json.RawMessage `json:"context_budget"`
	GenerationReserve json.RawMessage `json:"generation_reserve"`
	SystemPrompt      json.RawMessage `json:"system_prompt"`
	BearerTokenEnv    json.RawMessage `json:"bearer_token_env"`
}

// Load reads and validates a JSON config file.
func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultPath
	}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("config file %q not found; copy radichat.example.json or pass -config PATH", path)
		}
		return Config{}, fmt.Errorf("cannot read config file %q: %w; check permissions and pass -config PATH if needed", path, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return Config{}, fmt.Errorf("cannot stat config file %q: %w", path, err)
	}
	if info.IsDir() {
		return Config{}, fmt.Errorf("config path %q is a directory; pass a JSON file", path)
	}
	if info.Size() > MaxFileBytes {
		return Config{}, fmt.Errorf("config file %q exceeds the %d-byte limit; reduce the file size", path, MaxFileBytes)
	}

	data, err := io.ReadAll(io.LimitReader(f, MaxFileBytes+1))
	if err != nil {
		return Config{}, fmt.Errorf("cannot read config file %q: %w", path, err)
	}
	if int64(len(data)) > MaxFileBytes {
		return Config{}, fmt.Errorf("config file %q exceeds the %d-byte limit; reduce the file size", path, MaxFileBytes)
	}
	return Parse(path, data)
}

// Parse validates JSON config bytes. path is used only in error messages.
func Parse(path string, data []byte) (Config, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return Config{}, fmt.Errorf("config file %q is empty; provide a JSON object with %s", path, allowedFields)
	}
	if err := rejectDuplicateKeys(trimmed); err != nil {
		return Config{}, fmt.Errorf("config file %q: %w", path, err)
	}

	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.DisallowUnknownFields()
	var raw fileConfig
	if err := dec.Decode(&raw); err != nil {
		return Config{}, fmt.Errorf("config file %q: %s", path, jsonError(err))
	}
	if err := rejectTrailing(dec); err != nil {
		return Config{}, fmt.Errorf("config file %q: %w", path, err)
	}

	cfg := Config{Path: path}
	var err error
	cfg.Endpoint, err = parseRequiredString(raw.Endpoint, "endpoint")
	if err != nil {
		return Config{}, err
	}
	cfg.Model, err = parseRequiredString(raw.Model, "model")
	if err != nil {
		return Config{}, err
	}
	cfg.ContextBudget, err = parsePositiveInt(raw.ContextBudget, "context_budget")
	if err != nil {
		return Config{}, err
	}
	cfg.GenerationReserve, err = parsePositiveInt(raw.GenerationReserve, "generation_reserve")
	if err != nil {
		return Config{}, err
	}
	if cfg.GenerationReserve >= cfg.ContextBudget {
		return Config{}, fmt.Errorf("generation_reserve must be smaller than context_budget; lower generation_reserve or raise context_budget")
	}
	if raw.SystemPrompt != nil {
		cfg.SystemPrompt, err = parseOptionalString(raw.SystemPrompt, "system_prompt")
		if err != nil {
			return Config{}, err
		}
	}
	if raw.BearerTokenEnv != nil {
		cfg.BearerTokenEnv, err = parseRequiredString(raw.BearerTokenEnv, "bearer_token_env")
		if err != nil {
			return Config{}, err
		}
		if err := resolveBearerToken(&cfg); err != nil {
			return Config{}, err
		}
	}
	if err := validateEndpoint(cfg.Endpoint); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func parseRequiredString(raw json.RawMessage, name string) (string, error) {
	if raw == nil {
		return "", fmt.Errorf("missing %s; set %s in the config file", name, name)
	}
	s, err := parseString(raw, name)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(s) == "" {
		return "", fmt.Errorf("%s must be a nonempty string", name)
	}
	return s, nil
}

func parseOptionalString(raw json.RawMessage, name string) (string, error) {
	return parseString(raw, name)
}

func parseString(raw json.RawMessage, name string) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '"' {
		if string(trimmed) == "null" {
			return "", fmt.Errorf("%s must be a JSON string when present; omit the field instead of using null", name)
		}
		return "", fmt.Errorf("%s must be a JSON string", name)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("%s must be a JSON string", name)
	}
	return s, nil
}

func parsePositiveInt(raw json.RawMessage, name string) (int64, error) {
	if raw == nil {
		return 0, fmt.Errorf("missing %s; set a positive integer", name)
	}
	s := strings.TrimSpace(string(raw))
	if s == "" || s[0] == '"' || s[0] == '{' || s[0] == '[' || s == "null" || s == "true" || s == "false" {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	if strings.ContainsAny(s, ".eE+") {
		return 0, fmt.Errorf("%s must be a positive decimal integer without a fraction or exponent", name)
	}
	if len(s) > 1 && s[0] == '0' {
		return 0, fmt.Errorf("%s must be a positive integer without leading zeros", name)
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a positive integer fitting in 64 bits", name)
	}
	if n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return n, nil
}

func validateEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("endpoint is not a usable URL (%v); set an absolute http:// or https:// chat-completions URL", err)
	}
	if !u.IsAbs() || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("endpoint must be an absolute http:// or https:// URL; set the full chat-completions URL")
	}
	if u.Host == "" {
		return fmt.Errorf("endpoint is missing a host; set the full chat-completions URL")
	}
	if u.User != nil {
		return fmt.Errorf("endpoint must not contain userinfo/credentials; remove them from the URL")
	}
	if u.RawQuery != "" {
		return fmt.Errorf("endpoint must not contain a query string; put the full path in the URL without ?...")
	}
	if u.Fragment != "" || strings.Contains(endpoint, "#") {
		return fmt.Errorf("endpoint must not contain a fragment; remove #... from the URL")
	}
	if u.Opaque != "" {
		return fmt.Errorf("endpoint must be a normal http(s) URL, not an opaque URL")
	}
	return nil
}

func resolveBearerToken(cfg *Config) error {
	name := cfg.BearerTokenEnv
	if !validEnvName(name) {
		return fmt.Errorf("bearer_token_env must be a POSIX environment variable name (letters, digits, and underscore; not starting with a digit); fix the name in the config file")
	}
	val, ok := os.LookupEnv(name)
	if !ok {
		return fmt.Errorf("environment variable named by bearer_token_env is unset; export %s before starting RadiChat", name)
	}
	if val == "" {
		return fmt.Errorf("environment variable named by bearer_token_env is empty; export a nonempty token in %s", name)
	}
	if err := validateBearerToken(val); err != nil {
		return err
	}
	cfg.BearerToken = val
	return nil
}

func validEnvName(name string) bool {
	if name == "" || len(name) > maxEnvName {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c == '_':
		case i > 0 && c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return true
}

func validateBearerToken(token string) error {
	if len(token) > maxTokenBytes {
		return fmt.Errorf("environment variable named by bearer_token_env is not a usable HTTP header value; use a single-line token without control characters")
	}
	for i := 0; i < len(token); i++ {
		c := token[i]
		if c < 0x21 || c > 0x7E {
			return fmt.Errorf("environment variable named by bearer_token_env is not a usable HTTP header value; use a single-line token without control characters")
		}
	}
	return nil
}

func rejectTrailing(dec *json.Decoder) error {
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON after the config object; remove extra values")
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		var syn *json.SyntaxError
		if errors.As(err, &syn) {
			return fmt.Errorf("trailing JSON after the config object; remove extra values")
		}
		return fmt.Errorf("trailing JSON after the config object; remove extra values")
	}
	return nil
}

func jsonError(err error) string {
	var unk interface{ Error() string } = err
	msg := unk.Error()
	if strings.Contains(msg, "unknown field") {
		return fmt.Sprintf("%s; allowed fields are %s", msg, allowedFields)
	}
	var syn *json.SyntaxError
	if errors.As(err, &syn) {
		return fmt.Sprintf("malformed JSON at offset %d; fix the syntax", syn.Offset)
	}
	var typ *json.UnmarshalTypeError
	if errors.As(err, &typ) {
		if typ.Field != "" {
			return fmt.Sprintf("%s has the wrong JSON type; expected a JSON object with %s", typ.Field, allowedFields)
		}
		return fmt.Sprintf("config must be a JSON object with %s", allowedFields)
	}
	return fmt.Sprintf("malformed JSON (%s); fix the syntax", err)
}

func rejectDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	return consumeJSONValue(dec)
}

func consumeJSONValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("malformed JSON; fix the syntax")
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return fmt.Errorf("malformed JSON; fix the syntax")
			}
			key, ok := kt.(string)
			if !ok {
				return fmt.Errorf("malformed JSON object; fix the syntax")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON key %q; remove the extra occurrence", key)
			}
			seen[key] = struct{}{}
			if err := consumeJSONValue(dec); err != nil {
				return err
			}
		}
		_, err := dec.Token()
		if err != nil {
			return fmt.Errorf("malformed JSON; fix the syntax")
		}
		return nil
	case '[':
		for dec.More() {
			if err := consumeJSONValue(dec); err != nil {
				return err
			}
		}
		_, err := dec.Token()
		if err != nil {
			return fmt.Errorf("malformed JSON; fix the syntax")
		}
		return nil
	default:
		return fmt.Errorf("malformed JSON; fix the syntax")
	}
}
