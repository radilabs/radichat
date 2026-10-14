//go:build mockdiscovery

package discovery

import (
	"strings"
	"testing"

	"github.com/radilabs/radichat/internal/mockllm"
)

// TestP5AuthStartup runs A1-A7: authenticated and unauthenticated endpoint
// behavior, plus startup failures for an unusable bearer_token_env.
func TestP5AuthStartup(t *testing.T) {
	cases := []struct {
		spec  p5Spec
		check func(*p5Result) (bool, string)
	}{
		{
			spec: p5Spec{
				id: "A1", scenario: "auth-required-match", stdin: "hi\n",
				cfg:    p5Cfg(8192, 1024).withBearer("P5_SENTINEL_TOKEN"),
				env:    []p5EnvVar{{name: "P5_SENTINEL_TOKEN", value: mockllm.SentinelToken}},
				oracle: `docs/protocol.md#Request: when bearer_token_env is present, requests include "Authorization: Bearer <token>", and the authenticated path shares the zero-auth chat behavior.`,
				div: p5DivSpec{
					slug:     p5Slug("A1", "auth-required-match"),
					title:    "Case A1 (auth-required-match): authenticated request to a protected endpoint does not succeed as documented",
					expected: `docs/protocol.md#Request: a configured bearer credential is sent as "Authorization: Bearer <token>" and the endpoint then streams normally, so the turn completes with exit 0 and the assistant text on stdout.`,
					impact:   "The optional authenticated path is broken, so documented remote-endpoint usage fails.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 0 && r.stdout == "authorized\n" && len(r.records) == 1 &&
					r.records[0].Auth.String() == "match" &&
					r.records[0].Headers[mockllm.AuthorizationHeader] == mockllm.RedactedAuthorization
				return ok, p5Actual(r)
			},
		},
		{
			spec: p5Spec{
				id: "A2", scenario: "auth-required-missing", stdin: "hi\n",
				cfg:    p5Cfg(8192, 1024),
				oracle: `docs/protocol.md: "When bearer_token_env is omitted, Authorization is not sent", and docs/protocol.md#Incompatible-endpoints: endpoints that require a bearer token work only when bearer_token_env names a valid environment variable, so a zero-auth request to a protected endpoint fails clearly.`,
				div: p5DivSpec{
					slug:     p5Slug("A2", "auth-required-missing"),
					title:    "Case A2 (auth-required-missing): zero-auth request to a protected endpoint does not fail clearly as documented",
					expected: `docs/protocol.md: with no bearer_token_env configured, no Authorization header is sent and the 401 answer fails the turn clearly (exit 1 in EOF mode) with no retry.`,
					impact:   "A missing credential is not surfaced as a clear failure, so authentication problems are hidden from the operator.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 1 && len(r.records) == 1 && r.records[0].Auth.String() == "none" &&
					r.records[0].Headers[mockllm.AuthorizationHeader] == ""
				return ok, p5Actual(r)
			},
		},
		{
			spec: p5Spec{
				id: "A3", scenario: "auth-required-mismatch", stdin: "hi\n",
				cfg:    p5Cfg(8192, 1024).withBearer("P5_MISMATCH_TOKEN"),
				env:    []p5EnvVar{{name: "P5_MISMATCH_TOKEN", value: mockllm.SentinelMismatchToken}},
				oracle: `docs/protocol.md#Incompatible-endpoints: a wrong credential leaves the endpoint unanswered as a chat, so the turn fails clearly with no retry.`,
				div: p5DivSpec{
					slug:     p5Slug("A3", "auth-required-mismatch"),
					title:    "Case A3 (auth-required-mismatch): mismatched credential does not fail clearly as documented",
					expected: `docs/protocol.md: a mismatched bearer credential produces a 401 answer that fails the turn clearly (exit 1 in EOF mode) with exactly one request.`,
					impact:   "A wrong credential is not surfaced as a clear failure, so authentication problems are hidden from the operator.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 1 && len(r.records) == 1 && r.records[0].Auth.String() == "mismatch"
				return ok, p5Actual(r)
			},
		},
		{
			spec: p5Spec{
				id: "A4", scenario: "stream-success-basic", stdin: "hi\n",
				cfg:    p5Cfg(8192, 1024).withBearer("P5_UNSET_TOKEN"),
				oracle: `docs/cli.md#Exit codes: "Startup config errors always exit before the REPL and before any network access. That includes a missing, empty, or malformed token when bearer_token_env is set. Diagnostics do not print the token."`,
				div: p5DivSpec{
					slug:     p5Slug("A4", "unset-token-variable"),
					title:    "Case A4 (unset bearer_token_env variable): startup does not fail before network access as documented",
					expected: `docs/cli.md: an unset token variable is a startup config error: exit 1, no REPL, no network request, and no token value in the diagnostics.`,
					impact:   "A missing credential is not detected at startup, so the process contacts the network in an undefined auth state.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 1 && len(r.records) == 0 && r.stdout == "" && r.stderr != ""
				return ok, p5Actual(r)
			},
		},
		{
			spec: p5Spec{
				id: "A5", scenario: "stream-success-basic", stdin: "hi\n",
				cfg:    p5Cfg(8192, 1024).withBearer("P5_EMPTY_TOKEN"),
				env:    []p5EnvVar{{name: "P5_EMPTY_TOKEN", value: ""}},
				oracle: `docs/cli.md#Exit codes: "Startup config errors always exit before the REPL and before any network access. That includes a missing, empty, or malformed token when bearer_token_env is set. Diagnostics do not print the token."`,
				div: p5DivSpec{
					slug:     p5Slug("A5", "empty-token-value"),
					title:    "Case A5 (empty bearer token value): startup does not fail before network access as documented",
					expected: `docs/cli.md: an empty token value is a startup config error: exit 1, no REPL, and no network request.`,
					impact:   "An empty credential is not detected at startup, so the process contacts the network in an undefined auth state.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 1 && len(r.records) == 0 && r.stdout == "" && r.stderr != ""
				return ok, p5Actual(r)
			},
		},
		{
			spec: p5Spec{
				id: "A6", scenario: "stream-success-basic", stdin: "hi\n",
				cfg:    p5Cfg(8192, 1024).withBearer("P5_MALFORMED_TOKEN"),
				env:    []p5EnvVar{{name: "P5_MALFORMED_TOKEN", value: "not a valid token"}},
				oracle: `docs/cli.md#Exit codes: "Startup config errors always exit before the REPL and before any network access. That includes a missing, empty, or malformed token when bearer_token_env is set. Diagnostics do not print the token."`,
				div: p5DivSpec{
					slug:     p5Slug("A6", "malformed-token-value"),
					title:    "Case A6 (malformed bearer token value): startup does not fail before network access as documented",
					expected: `docs/cli.md: a token value that is not a usable single-line HTTP header value is a startup config error: exit 1, no REPL, no network request, and the value is not printed.`,
					impact:   "A malformed credential is not detected at startup, so the process contacts the network in an undefined auth state.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 1 && len(r.records) == 0 && r.stdout == "" && r.stderr != "" &&
					!strings.Contains(r.stderr, "not a valid token")
				return ok, p5Actual(r)
			},
		},
		{
			spec: p5Spec{
				id: "A7", scenario: "stream-success-basic", stdin: "hi\n",
				cfg:    p5Cfg(8192, 1024).withBearer("1BAD-NAME"),
				oracle: `docs/cli.md#Exit codes: "Startup config errors always exit before the REPL and before any network access", which covers an unusable bearer_token_env name.`,
				div: p5DivSpec{
					slug:     p5Slug("A7", "invalid-env-var-name"),
					title:    "Case A7 (invalid bearer_token_env name): startup does not fail before network access as documented",
					expected: `docs/cli.md: a bearer_token_env value that is not a POSIX environment variable name is a startup config error: exit 1, no REPL, and no network request.`,
					impact:   "An unusable variable name is not detected at startup, so the process contacts the network in an undefined auth state.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 1 && len(r.records) == 0 && r.stdout == "" && r.stderr != ""
				return ok, p5Actual(r)
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.spec.id, func(t *testing.T) {
			res := p5Run(t, tc.spec)
			ok, actual := tc.check(res)
			p5Conclude(t, tc.spec, res, ok, actual)
		})
	}
}

// TestP5EchoRedaction runs E1-E7: authenticated credential-echo redaction and
// unchanged zero-auth stream text.
func TestP5EchoRedaction(t *testing.T) {
	authCfg := p5Cfg(8192, 1024).withBearer("P5_SENTINEL_TOKEN")
	authEnv := []p5EnvVar{{name: "P5_SENTINEL_TOKEN", value: mockllm.SentinelToken}}
	cases := []struct {
		spec  p5Spec
		check func(*p5Result) (bool, string)
	}{
		{
			spec: p5Spec{
				id: "E1", scenario: "p5-echo-delta", stdin: "hi\n", cfg: authCfg, env: authEnv,
				oracle: `docs/protocol.md: "assistant stream text is filtered so a bearer token is not printed".`,
				div: p5DivSpec{
					slug:     p5Slug("E1", "p5-echo-delta"),
					title:    "Case E1 (token echo in one delta): the credential is printed in authenticated stream output",
					expected: `docs/protocol.md: a streamed delta containing the bearer token must not print it; ordinary surrounding text is kept.`,
					impact:   "The credential-redaction backstop fails, so a hostile or misbehaving endpoint can echo the operator's bearer token to the terminal.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 0 && r.stdout == "before [redacted] after\n" && len(r.records) == 1
				return ok, p5Actual(r)
			},
		},
		{
			spec: p5Spec{
				id: "E2", scenario: "p5-echo-split", stdin: "hi\n", cfg: authCfg, env: authEnv,
				oracle: `docs/protocol.md: the token is not printed "including when it is split across SSE deltas".`,
				div: p5DivSpec{
					slug:     p5Slug("E2", "p5-echo-split"),
					title:    "Case E2 (token split across deltas): the credential is printed in authenticated stream output",
					expected: `docs/protocol.md: a token split across SSE deltas must still be filtered; ordinary surrounding text is kept.`,
					impact:   "The credential-redaction backstop fails for fragmented echoes, so a hostile or misbehaving endpoint can print the operator's bearer token.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 0 && r.stdout == "before [redacted] after\n" && len(r.records) == 1
				return ok, p5Actual(r)
			},
		},
		{
			spec: p5Spec{
				id: "E3", scenario: "p5-echo-trailing-prefix", stdin: "hi\n", cfg: authCfg, env: authEnv,
				oracle: `docs/protocol.md: the token is not printed "when a successful stream ends on a proper prefix of the token. Ordinary surrounding text is kept."`,
				div: p5DivSpec{
					slug:     p5Slug("E3", "p5-echo-trailing-prefix"),
					title:    "Case E3 (trailing token prefix): a proper token prefix is printed in authenticated stream output",
					expected: `docs/protocol.md: when a successful stream ends on a proper prefix of the token, that prefix must not be printed; ordinary surrounding text is kept.`,
					impact:   "A partial credential leaks to the terminal at stream end, weakening the documented redaction backstop.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 0 && r.stdout == "keep this [redacted]\n" && len(r.records) == 1
				return ok, p5Actual(r)
			},
		},
		{
			spec: p5Spec{
				id: "E4", scenario: "p5-http-401-echo", stdin: "hi\n", cfg: authCfg, env: authEnv,
				notes:  "first of the two E4 inputs (401 body embedding the credential)",
				oracle: `docs/protocol.md: when bearer_token_env is present, "untrusted HTTP and stream error bodies are omitted".`,
				div: p5DivSpec{
					slug:     p5Slug("E4", "p5-http-401-echo"),
					title:    "Case E4 (401 body embedding the credential): the untrusted HTTP body is not omitted in authenticated mode",
					expected: `docs/protocol.md: in authenticated mode the non-success HTTP body is omitted; only the status classification and a fix hint are shown, so the credential cannot be echoed through an error body.`,
					impact:   "A hostile or misbehaving endpoint can echo the operator's bearer token through an HTTP error body.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 1 && len(r.records) == 1 && !p5ContainsSentinel(r.stdout) && !p5ContainsSentinel(r.stderr)
				return ok, p5Actual(r)
			},
		},
		{
			spec: p5Spec{
				id: "E4", scenario: "p5-http-500-echo", stdin: "hi\n", cfg: authCfg, env: authEnv,
				notes:  "second of the two E4 inputs (500 body embedding the credential)",
				oracle: `docs/protocol.md: when bearer_token_env is present, "untrusted HTTP and stream error bodies are omitted".`,
				div: p5DivSpec{
					slug:     p5Slug("E4", "p5-http-500-echo"),
					title:    "Case E4 (500 body embedding the credential): the untrusted HTTP body is not omitted in authenticated mode",
					expected: `docs/protocol.md: in authenticated mode the non-success HTTP body is omitted; only the status classification and a fix hint are shown, so the credential cannot be echoed through an error body.`,
					impact:   "A hostile or misbehaving endpoint can echo the operator's bearer token through an HTTP error body.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 1 && len(r.records) == 1 && !p5ContainsSentinel(r.stdout) && !p5ContainsSentinel(r.stderr)
				return ok, p5Actual(r)
			},
		},
		{
			spec: p5Spec{
				id: "E5", scenario: "p5-stream-error-echo", stdin: "hi\n", cfg: authCfg, env: authEnv,
				oracle: `docs/protocol.md: when bearer_token_env is present, "untrusted HTTP and stream error bodies are omitted; status or stream classification and a fix hint are kept".`,
				div: p5DivSpec{
					slug:     p5Slug("E5", "p5-stream-error-echo"),
					title:    "Case E5 (stream error object embedding the credential): the untrusted stream error payload is not omitted in authenticated mode",
					expected: `docs/protocol.md: in authenticated mode a stream error payload is omitted; only the stream classification and a fix hint are shown, so the credential cannot be echoed through a stream error object.`,
					impact:   "A hostile or misbehaving endpoint can echo the operator's bearer token through a stream error object.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 1 && len(r.records) == 1 && !p5ContainsSentinel(r.stdout) && !p5ContainsSentinel(r.stderr)
				return ok, p5Actual(r)
			},
		},
		{
			spec: p5Spec{
				id: "E6", scenario: "p5-redirect-echo", stdin: "hi\n", cfg: authCfg, env: authEnv,
				oracle: `docs/protocol.md: when bearer_token_env is present, transport failures are classified "without quoting the underlying error or redirect Location, which can echo or re-encode a token".`,
				div: p5DivSpec{
					slug:     p5Slug("E6", "p5-redirect-echo"),
					title:    "Case E6 (redirect Location embedding the credential): the refused Location is quoted in authenticated mode",
					expected: `docs/protocol.md: in authenticated mode a refused redirect is classified without quoting the redirect Location, so a credential-bearing Location cannot be echoed.`,
					impact:   "A hostile or misbehaving endpoint can echo or re-encode the operator's bearer token through a redirect Location.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 1 && len(r.records) == 1 && !p5ContainsSentinel(r.stdout) && !p5ContainsSentinel(r.stderr)
				return ok, p5Actual(r)
			},
		},
		{
			spec: p5Spec{
				id: "E7", scenario: "p5-echo-delta", stdin: "hi\n", cfg: p5Cfg(8192, 1024),
				oracle: `docs/protocol.md: "Zero-auth stream text is unchanged."`,
				div: p5DivSpec{
					slug:     p5Slug("E7", "p5-echo-delta-zero-auth"),
					title:    "Case E7 (same echo scenario in zero-auth mode): stream text is not unchanged as documented",
					expected: `docs/protocol.md: with no bearer_token_env configured, streamed assistant text is unchanged — the same payload prints verbatim.`,
					impact:   "Zero-auth output is altered even though the documentation promises it is unchanged.",
					severity: "major",
				},
			},
			check: func(r *p5Result) (bool, string) {
				ok := r.exit == 0 && r.stdout == "before "+mockllm.SentinelToken+" after\n" && len(r.records) == 1
				return ok, p5Actual(r)
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.spec.id, func(t *testing.T) {
			res := p5Run(t, tc.spec)
			ok, actual := tc.check(res)
			p5Conclude(t, tc.spec, res, ok, actual)
		})
	}
}
