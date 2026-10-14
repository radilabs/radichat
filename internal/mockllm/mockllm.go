// Package mockllm provides a deterministic, standard-library-only mock of an
// OpenAI-compatible Chat Completions endpoint for testing RadiChat.
//
// The mock serves named scenarios that reproduce the documented protocol
// behavior in docs/protocol.md: successful streamed responses, SSE framing and
// exact configured byte segmentation, malformed or incomplete responses, HTTP
// errors, bearer-token match/missing/mismatch, refused redirects, and
// request/context-accounting fixtures.
//
// A scenario is data, not code: the Scenario struct declares the status,
// headers, redirect target, authentication expectation, response wire bytes,
// and delivery mode. The same scenario therefore produces the same bytes on
// every run and can be listed, documented, and inspected.
//
// Two delivery modes exist. ModeHTTP writes through net/http's ResponseWriter,
// which produces normal flushed, chunked responses. ModeRaw hijacks the
// connection and writes the exact configured bytes, which is required when a
// test must control response framing byte for byte.
//
// Every request is recorded. Recorded observations never include a
// bearer-token value: the recorder stores only whether a bearer credential was
// present, whether it was malformed, and whether it matched. Nothing in this
// package reads or writes files, and no third-party dependency is used.
//
// The mock binds loopback only. See Options and Start.
package mockllm

import (
	"fmt"
	"net/http"
	"sort"
)

// Mode selects how a scenario's response bytes are delivered.
type Mode int

const (
	// ModeHTTP writes through net/http's ResponseWriter. Responses are
	// flushed per segment and framed by net/http (chunked transfer encoding
	// for HTTP/1.1). Use it for ordinary traffic.
	ModeHTTP Mode = iota

	// ModeRaw hijacks the connection and writes the configured bytes
	// verbatim, terminated by closing the connection. Use it when a test
	// needs exact control over response bytes and framing.
	ModeRaw
)

// String renders the mode name.
func (m Mode) String() string {
	switch m {
	case ModeHTTP:
		return "http"
	case ModeRaw:
		return "raw"
	default:
		return fmt.Sprintf("Mode(%d)", int(m))
	}
}

// AuthSpec declares the bearer-token behavior of a scenario.
type AuthSpec struct {
	// Required makes the scenario answer 401 unless the request carries a
	// bearer credential that matches Token.
	Required bool

	// Token is the expected bearer credential. When empty and Required is
	// true, any nonempty bearer credential is accepted.
	Token string
}

// Expect declares the request shape a scenario is designed to observe.
//
// The mock compares the recorded request against Expect and reports the result
// per request. It never rejects a non-conforming request, because the mock's
// job is to record what a client actually sent.
type Expect struct {
	// Method is the expected HTTP method, for example "POST".
	Method string

	// Fields are expected JSON body values keyed by top-level field name.
	// Only the listed fields are compared.
	Fields map[string]any

	// Messages are expected message roles in order. The count is compared
	// and each listed role must appear at that index.
	Messages []string

	// Note explains what the fixture demonstrates. It is documentation.
	Note string
}

// Scenario is one named, deterministic endpoint behavior.
//
// The zero value is not usable; register scenarios with Register or build them
// with the helpers in scenarios.go.
type Scenario struct {
	// Name is the stable identifier used on the command line and in URL
	// paths, for example "stream-success-basic".
	Name string

	// Family groups scenarios for listing and documentation, for example
	// "stream" or "auth".
	Family string

	// Description explains what behavior the scenario reproduces.
	Description string

	// Auth declares the bearer-token expectation.
	Auth AuthSpec

	// Status is the HTTP status code written before the body. Zero means 200.
	Status int

	// Headers are additional response headers.
	Headers map[string]string

	// Redirect, when nonempty, is written as the Location header together
	// with Status (default 302). A redirect response has no body.
	Redirect string

	// Wire is the exact response body, delivered one Segment per write.
	Wire []Segment

	// Mode selects the delivery mechanism.
	Mode Mode

	// Expect declares the request shape this scenario observes.
	Expect Expect
}

// Path returns the URL path that selects this scenario.
func (s Scenario) Path() string {
	return scenarioPathPrefix + s.Name
}

// status returns the effective HTTP status code.
func (s Scenario) status() int {
	if s.Status == 0 {
		return http.StatusOK
	}
	return s.Status
}

// redirectStatus returns the effective status for a redirect scenario.
func (s Scenario) redirectStatus() int {
	if s.Status == 0 {
		return http.StatusFound
	}
	return s.Status
}

// registry maps scenario names to scenarios.
var registry = map[string]Scenario{}

// Register adds a scenario to the catalog. It panics on a duplicate name so a
// copy-paste error cannot silently shadow a scenario.
func Register(s Scenario) {
	if s.Name == "" {
		panic("mockllm: scenario has no name")
	}
	if _, dup := registry[s.Name]; dup {
		panic("mockllm: duplicate scenario name " + s.Name)
	}
	registry[s.Name] = s
}

// Catalog returns every registered scenario in sorted name order.
func Catalog() []Scenario {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]Scenario, 0, len(names))
	for _, name := range names {
		out = append(out, registry[name])
	}
	return out
}

// Lookup returns the named scenario.
func Lookup(name string) (Scenario, bool) {
	s, ok := registry[name]
	return s, ok
}

// Names returns every registered scenario name in sorted order.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Families returns every scenario family in sorted order.
func Families() []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range registry {
		if !seen[s.Family] {
			seen[s.Family] = true
			out = append(out, s.Family)
		}
	}
	sort.Strings(out)
	return out
}
