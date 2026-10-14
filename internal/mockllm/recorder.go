package mockllm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// maxRequestBody bounds how much of a request body the mock reads and records.
// It exists so a malformed client cannot make the mock allocate without bound.
const maxRequestBody = 1 << 20

// maxHeaderRecord bounds how many bytes of any single recorded header value are
// retained. It keeps recorded observations small and reviewable.
const maxHeaderRecord = 256

// AuthorizationHeader is the header name the mock inspects. It is exported so
// tests can assert against it.
const AuthorizationHeader = "Authorization"

// bearerPrefix is the only accepted credential scheme.
const bearerPrefix = "Bearer "

// Record is one recorded request observation.
//
// A Record never contains a credential value. The Authorization header is
// reduced to a constant placeholder, Cookie is omitted, and the query string is
// not captured at all, because a query string can carry tokens such as
// ?access_token=... Auth describes the authentication decision in coarse terms:
// whether a credential was present, whether it was malformed, and whether it
// matched. That is enough to assert authentication behavior without retaining
// the credential.
type Record struct {
	// Method is the HTTP method.
	Method string

	// Path is the URL path, without the query string.
	Path string

	// Headers holds selected request headers. The Authorization header is
	// recorded as RedactedAuthorization, never as its value. Cookie is
	// omitted entirely.
	Headers map[string]string

	// Body holds the decoded JSON request body, or nil when the body was not
	// a JSON object.
	Body map[string]any

	// BodyRaw is the raw request body text. A request body never carries the
	// bearer token, which travels in the Authorization header, so recording
	// it does not retain a credential.
	BodyRaw string

	// Auth is the classified authentication outcome.
	Auth AuthOutcome

	// Scenario is the name of the scenario that handled the request.
	Scenario string

	// MatchedExpect reports whether the request matched the scenario's
	// declared Expect, and why not.
	ExpectCheck ExpectResult
}

// AuthOutcome records how the mock classified the request's bearer credential,
// without storing the credential.
type AuthOutcome struct {
	// Present reports whether an Authorization header was supplied.
	Present bool

	// Malformed reports whether the header was present but not a usable
	// "Bearer <token>" value.
	Malformed bool

	// Matched reports whether a supplied credential matched the scenario's
	// expected token. When no expected token is configured, any nonempty
	// credential counts as matched.
	Matched bool
}

// String renders the outcome without any credential material.
func (a AuthOutcome) String() string {
	switch {
	case a.Malformed:
		return "malformed"
	case !a.Present:
		return "none"
	case a.Matched:
		return "match"
	default:
		return "mismatch"
	}
}

// ExpectResult reports how a recorded request compared to a scenario's Expect.
type ExpectResult struct {
	// Matched reports whether every declared expectation held.
	Matched bool

	// Problems lists each mismatch in human-readable form. It never contains
	// credential material.
	Problems []string
}

// RedactedAuthorization is the placeholder written in place of an
// Authorization header value. It is deliberately not a real-looking token so a
// scan for credential patterns cannot mistake it for one.
const RedactedAuthorization = "<redacted-authorization>"

// classifyAuth classifies an Authorization header without retaining it.
func classifyAuth(header, expected string) AuthOutcome {
	if header == "" {
		return AuthOutcome{}
	}
	token, ok := strings.CutPrefix(header, bearerPrefix)
	if !ok || token == "" {
		return AuthOutcome{Present: true, Malformed: true}
	}
	if expected == "" {
		return AuthOutcome{Present: true, Matched: true}
	}
	return AuthOutcome{Present: true, Matched: token == expected}
}

// recordedHeaders copies the headers worth asserting on.
//
// Authorization is reduced to a constant placeholder and Cookie is omitted
// entirely: both can carry credentials, so neither value may reach a Record.
// The remaining headers are transport metadata that RadiChat sets from its
// configuration, not secret material.
//
// A caller that needs to assert "no cookie was sent" should check that the
// key is absent, which is exactly what omitting it preserves.
func recordedHeaders(h http.Header) map[string]string {
	out := make(map[string]string, 3)
	if v := h.Get("Content-Type"); v != "" {
		out["Content-Type"] = truncate(v)
	}
	if v := h.Get("Accept"); v != "" {
		out["Accept"] = truncate(v)
	}
	if h.Get(AuthorizationHeader) != "" {
		out[AuthorizationHeader] = RedactedAuthorization
	}
	return out
}

func truncate(s string) string {
	if len(s) > maxHeaderRecord {
		return s[:maxHeaderRecord] + "..."
	}
	return s
}

// decodeBody decodes a JSON object request body. Anything else yields nil.
func decodeBody(body []byte) map[string]any {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return nil
	}
	return m
}

// checkExpect compares a recorded request against a scenario's expectation.
func checkExpect(sc Scenario, rec *Record) ExpectResult {
	res := ExpectResult{Matched: true}
	if sc.Expect.Method != "" && rec.Method != sc.Expect.Method {
		res.Matched = false
		res.Problems = append(res.Problems, "method "+rec.Method+" != "+sc.Expect.Method)
	}
	for k, want := range sc.Expect.Fields {
		got, ok := rec.Body[k]
		if !ok {
			res.Matched = false
			res.Problems = append(res.Problems, "missing body field "+k)
			continue
		}
		if !valuesEqual(got, want) {
			res.Matched = false
			res.Problems = append(res.Problems, "body field "+k+" mismatch")
		}
	}
	if len(sc.Expect.Messages) > 0 {
		gotRoles := messageRoles(rec.Body)
		if len(gotRoles) != len(sc.Expect.Messages) {
			res.Matched = false
			res.Problems = append(res.Problems, "message count mismatch")
		} else {
			for i, want := range sc.Expect.Messages {
				if gotRoles[i] != want {
					res.Matched = false
					res.Problems = append(res.Problems, "message role mismatch at index "+strconv.Itoa(i))
				}
			}
		}
	}
	return res
}

// messageRoles extracts the roles from a decoded request body's messages array.
func messageRoles(body map[string]any) []string {
	raw, ok := body["messages"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			out = append(out, "")
			continue
		}
		role, _ := m["role"].(string)
		out = append(out, role)
	}
	return out
}

// valuesEqual compares a decoded JSON value against an expected literal,
// normalizing numbers so an integer expectation matches a decoded float64.
func valuesEqual(got, want any) bool {
	switch w := want.(type) {
	case int:
		f, ok := got.(float64)
		return ok && f == float64(w)
	case int64:
		f, ok := got.(float64)
		return ok && f == float64(w)
	case float64:
		f, ok := got.(float64)
		return ok && f == w
	case bool:
		b, ok := got.(bool)
		return ok && b == w
	case string:
		s, ok := got.(string)
		return ok && s == w
	default:
		return false
	}
}

// errBodyTooLarge is returned when a request body exceeds maxRequestBody.
var errBodyTooLarge = errors.New("mockllm: request body too large")

// readRequestBody reads and records a request body.
func readRequestBody(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", nil
	}
	br := bufio.NewReader(io.LimitReader(r.Body, maxRequestBody+1))
	var sb strings.Builder
	n, err := io.Copy(&sb, br)
	if err != nil {
		return "", err
	}
	if n > maxRequestBody {
		return "", errBodyTooLarge
	}
	return sb.String(), nil
}
