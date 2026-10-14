//go:build mockdiscovery

package discovery

import (
	"fmt"
	"strings"
	"testing"

	"github.com/radilabs/radichat/internal/mockllm"
)

// p5TableCase is one must-run matrix row from tasks/phase-5.md. kind is
// "success" or "failure"; want is the exact stdout the REPL must produce for a
// success (the REPL appends one trailing newline when the streamed text does
// not already end in one); wantStderrContains is a documented stderr substring
// that must be present.
type p5TableCase struct {
	id                 string
	scenario           string
	stdin              string
	kind               string
	want               string
	wantStderrContains string
	oracle             string
}

func p5SuccessSpec(tc p5TableCase) p5Spec {
	return p5Spec{
		id:       tc.id,
		scenario: tc.scenario,
		stdin:    tc.stdin,
		oracle:   tc.oracle,
		div: p5DivSpec{
			slug:     p5Slug(tc.id, tc.scenario),
			title:    "Case " + tc.id + " (" + tc.scenario + "): documented successful stream behavior not reproduced",
			expected: tc.oracle + " The turn must therefore complete: exit 0 (docs/cli.md#Exit codes: EOF after a session with no unrecovered turn failure), the exact assistant text on stdout, and exactly one recorded request.",
			impact:   "A stream shape the documentation says RadiChat supports fails or is mis-transcribed against the built binary, so a compliant endpoint is rejected or its output is corrupted.",
			severity: "major",
		},
	}
}

func p5FailureSpec(tc p5TableCase) p5Spec {
	return p5Spec{
		id:       tc.id,
		scenario: tc.scenario,
		stdin:    tc.stdin,
		oracle:   tc.oracle,
		div: p5DivSpec{
			slug:     p5Slug(tc.id, tc.scenario),
			title:    "Case " + tc.id + " (" + tc.scenario + "): documented failure behavior not reproduced",
			expected: tc.oracle + " The turn must therefore fail clearly: exit 1 in EOF mode (docs/cli.md#Exit codes), a diagnostic on stderr, exactly one request (docs/protocol.md: failed requests are not retried), and no hidden fallback.",
			impact:   "RadiChat accepts or silently mishandles a response the documentation says must fail, so an incompatible endpoint is treated as compatible.",
			severity: "major",
		},
	}
}

func p5CheckTable(tc p5TableCase, r *p5Result) (bool, string) {
	if tc.kind == "success" {
		ok := r.exit == 0 && r.stdout == tc.want+"\n" && len(r.records) == 1 &&
			r.records[0].Method == "POST" && !r.records[0].Auth.Present
		return ok, p5Actual(r)
	}
	ok := r.exit == 1 && r.stderr != "" && len(r.records) == 1
	actual := p5Actual(r)
	if ok && tc.wantStderrContains != "" && !strings.Contains(r.stderr, tc.wantStderrContains) {
		ok = false
		actual += "; stderr is missing the documented substring " + fmt.Sprintf("%q", tc.wantStderrContains)
	}
	return ok, actual
}

func p5RunTable(t *testing.T, cases []p5TableCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			var spec p5Spec
			if tc.kind == "success" {
				spec = p5SuccessSpec(tc)
			} else {
				spec = p5FailureSpec(tc)
			}
			res := p5Run(t, spec)
			ok, actual := p5CheckTable(tc, res)
			p5Conclude(t, spec, res, ok, actual)
		})
	}
}

// TestP5HarnessSelfCheck is a negative control that proves the verdict logic
// can actually detect a divergence. It reuses a real successful result and
// asserts that deliberately wrong expectations are rejected, so a green matrix
// cannot be vacuous.
func TestP5HarnessSelfCheck(t *testing.T) {
	spec := p5Spec{
		id: "SELF", scenario: "stream-success-basic", stdin: "hello\n",
		oracle: "harness self-check: proves PASS/DIVERGENCE classification bites",
	}
	res := p5Run(t, spec)
	if res.exit != 0 || res.stdout != "hello\n" {
		t.Fatalf("self-check baseline did not pass: %s", p5Actual(res))
	}
	wrongOutput := p5TableCase{id: "SELF", scenario: "stream-success-basic", stdin: "hello\n", kind: "success", want: "definitely-not-the-output"}
	if ok, _ := p5CheckTable(wrongOutput, res); ok {
		t.Fatal("self-check failed: a wrong stdout expectation was accepted as PASS")
	}
	wrongKind := p5TableCase{id: "SELF", scenario: "stream-success-basic", stdin: "hello\n", kind: "failure"}
	if ok, _ := p5CheckTable(wrongKind, res); ok {
		t.Fatal("self-check failed: a successful turn was accepted as the expected failure")
	}
	p5EvidenceLine(spec, res, "PASS",
		"harness self-check: baseline passed, and deliberately wrong stdout and failure expectations were both rejected by the verdict logic")
}

// TestP5StreamSuccess runs S1-S8: documented successful SSE parsing.
func TestP5StreamSuccess(t *testing.T) {
	p5RunTable(t, []p5TableCase{
		{id: "S1", scenario: "stream-success-basic", stdin: "hello\n", kind: "success", want: "hello",
			oracle: `docs/protocol.md#Stream: "data: [DONE]" is the terminal marker and "assistant deltas are written to stdout as they arrive".`},
		{id: "S2", scenario: "stream-success-multibyte", stdin: "hi\n", kind: "success", want: "héllo🌍日本語",
			oracle: `docs/protocol.md#Stream: the decoder accepts "UTF-8 text in JSON string deltas, including multibyte characters".`},
		{id: "S3", scenario: "stream-success-role-only", stdin: "hi\n", kind: "success", want: "ok",
			oracle: `docs/protocol.md#Stream: the decoder accepts "role-only chunks and empty deltas".`},
		{id: "S4", scenario: "stream-success-crlf", stdin: "hi\n", kind: "success", want: "crlf text",
			oracle: `docs/protocol.md#Stream: the decoder accepts "LF and CRLF line endings".`},
		{id: "S5", scenario: "stream-success-comments", stdin: "hi\n", kind: "success", want: "commented",
			oracle: `docs/protocol.md#Stream: the decoder accepts "SSE comments (: lines) and ignored event / id / retry fields".`},
		{id: "S7", scenario: "stream-success-empty-content", stdin: "hi\n", kind: "success", want: "",
			oracle: `docs/protocol.md#Stream: "role-only chunks and empty deltas" are accepted, so a completion carrying no content is a success, not an error.`},
		{id: "S8", scenario: "stream-success-length-finish", stdin: "hi\n", kind: "success", want: "truncated text",
			oracle: `docs/protocol.md#Stream: finish_reason "length" is a success — "RadiChat still commits the complete streamed text and does not locally truncate".`},
	})
}

// TestP5OmittedContentType runs S6. The accepted catalog scenario
// stream-success-no-content-type cannot exercise the documented branch: its
// HTTP-mode delivery lets net/http sniff "Content-Type: text/plain;
// charset=utf-8" into the response (verified with curl during this phase), so
// a run against the accepted payload only demonstrates the documented
// rejection of a present-but-wrong content type. The documented
// omitted-content-type branch is exercised with the harness-local raw payload
// registered in p5_scenarios_test.go, which the Phase 5 scope explicitly
// permits when the accepted catalog lacks a needed payload.
func TestP5OmittedContentType(t *testing.T) {
	t.Run("accepted-payload", func(t *testing.T) {
		accepted := p5Spec{
			id: "S6", scenario: "stream-success-no-content-type", stdin: "hi\n",
			oracle: `docs/protocol.md#Stream: "Successful responses must be HTTP 2xx with Content-Type text/event-stream (or an omitted content type, which is still parsed as SSE)."`,
		}
		resAccepted := p5Run(t, accepted)
		p5NotRun(t, accepted, resAccepted,
			`the accepted payload cannot omit Content-Type: the mock's HTTP mode lets net/http sniff "text/plain; charset=utf-8" into the response (observed here and verified with curl), so this run exercised the documented present-but-wrong-content-type rejection instead of the omitted-content-type branch`)
	})
	t.Run("raw-payload", func(t *testing.T) {
		raw := p5Spec{
			id: "S6", scenario: "p5-no-content-type-raw", stdin: "hi\n",
			notes:  "harness-local raw payload that truly omits Content-Type",
			oracle: `docs/protocol.md#Stream: "Successful responses must be HTTP 2xx with Content-Type text/event-stream (or an omitted content type, which is still parsed as SSE)."`,
			div: p5DivSpec{
				slug:     p5Slug("S6", "p5-no-content-type-raw"),
				title:    "Case S6 (omitted Content-Type): a response with no Content-Type is not parsed as SSE",
				expected: `docs/protocol.md#Stream: an omitted content type is still parsed as SSE, so the turn completes with exit 0 and the exact assistant text on stdout.`,
				impact:   "A compliant endpoint that omits Content-Type is rejected, so documented behavior fails.",
				severity: "major",
			},
		}
		resRaw := p5Run(t, raw)
		ok := !resRaw.timedOut && resRaw.exit == 0 && resRaw.stdout == "untyped\n" && len(resRaw.records) == 1
		p5Conclude(t, raw, resRaw, ok, p5Actual(resRaw))
	})
}

// TestP5Framing runs F1-F6: fixed-fragment SSE parsing.
func TestP5Framing(t *testing.T) {
	p5RunTable(t, []p5TableCase{
		{id: "F1", scenario: "fragment-utf8-inside-rune", stdin: "hi\n", kind: "success", want: "héllo",
			oracle: `docs/protocol.md#Stream: the decoder accepts "fragmented TCP reads" and "UTF-8 text in JSON string deltas", so a rune split across writes is reassembled.`},
		{id: "F2", scenario: "fragment-inside-crlf", stdin: "hi\n", kind: "success", want: "crlf",
			oracle: `docs/protocol.md#Stream: the decoder accepts "fragmented TCP reads" and "LF and CRLF line endings", so a CRLF split across writes is not mistaken for two line endings.`},
		{id: "F3", scenario: "fragment-inside-data-field", stdin: "hi\n", kind: "success", want: "frag",
			oracle: `docs/protocol.md#Stream: the decoder accepts "fragmented TCP reads", so a data: field split across writes is reassembled before parsing.`},
		{id: "F4", scenario: "fragment-inside-done", stdin: "hi\n", kind: "success", want: "done",
			oracle: `docs/protocol.md#Stream: "data: [DONE]" is the terminal marker and fragmented TCP reads are accepted, so a split marker is still recognized.`},
		{id: "F5", scenario: "fragment-byte-by-byte", stdin: "hi\n", kind: "success", want: "héllo",
			oracle: `docs/protocol.md#Stream: the decoder accepts "fragmented TCP reads", so one byte per write still assembles the same assistant text.`},
		{id: "F6", scenario: "fragment-multiple-events-one-write", stdin: "hi\n", kind: "success", want: "onetwo",
			oracle: `docs/protocol.md#Stream: the decoder accepts "fragmented TCP reads, including multiple events in one read".`},
	})
}

// TestP5Malformed runs M1-M12: malformed streams, supported stream-error and
// finish handling.
func TestP5Malformed(t *testing.T) {
	p5RunTable(t, []p5TableCase{
		{id: "M1", scenario: "malformed-json-event", stdin: "hi\n", kind: "failure",
			oracle: `docs/protocol.md#Stream: "malformed events ... fail".`},
		{id: "M2", scenario: "malformed-delta-content-not-string", stdin: "hi\n", kind: "failure",
			oracle: `docs/protocol.md#Stream: "delta.content must be a JSON string when present".`},
		{id: "M3", scenario: "malformed-multiple-choices", stdin: "hi\n", kind: "failure",
			oracle: `docs/protocol.md#Stream: "multiple choices ... fail".`},
		{id: "M4", scenario: "malformed-early-termination", stdin: "hi\n", kind: "failure",
			oracle: `docs/protocol.md#Stream: "Premature EOF without [DONE] ... fail".`},
		{id: "M5", scenario: "malformed-early-termination-mid-event", stdin: "hi\n", kind: "failure",
			oracle: `docs/protocol.md#Stream: "Premature EOF without [DONE] ... fail"; truncation inside an event is still a premature EOF.`},
		{id: "M6", scenario: "stream-error-object", stdin: "hi\n", kind: "failure",
			oracle: `docs/protocol.md#Stream: "Stream JSON error objects fail as endpoint errors".`},
		{id: "M7", scenario: "stream-error-non-object", stdin: "hi\n", kind: "failure",
			oracle: `docs/protocol.md#Stream: "A non-null error value that is not a JSON object (array, bool, string, number) fails as a malformed stream error payload".`},
		{id: "M8", scenario: "stream-tool-calls-rejected", stdin: "hi\n", kind: "failure",
			oracle: `docs/protocol.md#Stream: "Tool-call / function-call deltas are errors".`},
		{id: "M9", scenario: "stream-function-call-rejected", stdin: "hi\n", kind: "failure",
			oracle: `docs/protocol.md#Stream: "Tool-call / function-call deltas are errors".`},
		{id: "M10", scenario: "stream-finish-content-filter", stdin: "hi\n", kind: "failure",
			oracle: `docs/protocol.md#Stream: failure finish reasons are "tool_calls, function_call, content_filter, or any other nonempty reason".`},
		{id: "M11", scenario: "stream-finish-unsupported", stdin: "hi\n", kind: "failure",
			oracle: `docs/protocol.md#Stream: failure finish reasons include "any other nonempty reason".`},
		{id: "M12", scenario: "stream-error-null-ignored", stdin: "hi\n", kind: "success", want: "fine",
			oracle: `docs/protocol.md#Stream: "JSON error: null or an omitted error field is not an error".`},
	})
}

// TestP5HTTPErrors runs H1-H6: HTTP error statuses and non-SSE responses.
//
// Dr Watson correction: H1-H4 additionally require the documented zero-auth
// bounded snippet to appear, using each fixture's own body text, so the cases
// assert the documented diagnostic content and not only that some error
// occurred. Exactly one request (no retry) and exit 1 remain required; other
// diagnostic wording stays advisory.
func TestP5HTTPErrors(t *testing.T) {
	p5RunTable(t, []p5TableCase{
		{id: "H1", scenario: "http-error-401", stdin: "hi\n", kind: "failure",
			wantStderrContains: "unauthorized",
			oracle:             `docs/protocol.md: non-success responses fail clearly with no retry, and in zero-auth mode "non-success HTTP bodies ... are shown as a bounded diagnostic snippet" — this fixture's body is "unauthorized".`},
		{id: "H2", scenario: "http-error-429", stdin: "hi\n", kind: "failure",
			wantStderrContains: "rate limited",
			oracle:             `docs/protocol.md: non-success responses fail clearly with no retry, and in zero-auth mode the non-success body is shown as a bounded diagnostic snippet — this fixture's body carries "rate limited".`},
		{id: "H3", scenario: "http-error-500", stdin: "hi\n", kind: "failure",
			wantStderrContains: "internal error",
			oracle:             `docs/protocol.md: non-success responses fail clearly with no retry, and in zero-auth mode the non-success body is shown as a bounded diagnostic snippet — this fixture's body carries "internal error".`},
		{id: "H4", scenario: "http-error-404-unknown-model", stdin: "hi\n", kind: "failure",
			wantStderrContains: "model not found",
			oracle:             `docs/protocol.md: non-success responses fail clearly with no retry, and in zero-auth mode the non-success body is shown as a bounded diagnostic snippet — this fixture's body carries "model not found".`},
		{id: "H5", scenario: "non-sse-content-type", stdin: "hi\n", kind: "failure",
			oracle: `docs/protocol.md#Stream: "Successful responses must be HTTP 2xx with Content-Type text/event-stream ... Other content types fail".`},
		{id: "H6", scenario: "non-sse-empty-body", stdin: "hi\n", kind: "failure",
			oracle: `docs/protocol.md#Stream: "Premature EOF without [DONE] ... fail".`},
	})
}

// TestP5Redirects runs R1-R3: redirect refusal with exactly one request and
// the documented zero-auth Location diagnostic.
//
// Dr Watson correction for R3: the accepted redirect-absolute scenario targets
// 127.0.0.1:1, where a followed redirect fails to dial and produces no second
// record, so "no second record" cannot distinguish refusal from a dial
// failure, and a dial error can quote the URL. The R3 stimulus is therefore a
// harness-local scenario whose absolute Location points at this same live
// mock server's stream-success-basic route: a follow would add a second
// recorded POST and a success, so refusal is directly observable.
func TestP5Redirects(t *testing.T) {
	p5RunTable(t, []p5TableCase{
		{id: "R1", scenario: "redirect-to-final-url", stdin: "hi\n", kind: "failure",
			wantStderrContains: "/v1/chat/completions/stream-success-basic",
			oracle:             `docs/protocol.md: "Redirects are refused" and "redirect refusals include the refused Location (redacted)" in zero-auth mode.`},
		{id: "R2", scenario: "redirect-with-query", stdin: "hi\n", kind: "failure",
			wantStderrContains: "token=placeholder",
			oracle:             `docs/protocol.md: "Redirects are refused" and the refused Location, including its query string, is included in the zero-auth diagnostic.`},
	})
	t.Run("R3", func(t *testing.T) {
		registerP5RedirectAbsoluteLive()
		sc, ok := mockllm.Lookup("p5-redirect-absolute-live")
		if !ok {
			t.Fatal("p5-redirect-absolute-live was not registered")
		}
		spec := p5Spec{
			id: "R3", scenario: "p5-redirect-absolute-live", stdin: "hi\n",
			oracle: `docs/protocol.md: "Redirects are refused" and the refused absolute Location is included in the zero-auth diagnostic; "Failed requests are not retried".`,
			div: p5DivSpec{
				slug:     p5Slug("R3", "p5-redirect-absolute-live"),
				title:    "Case R3 (absolute redirect to a live route): the redirect is followed, or the refusal is not observable",
				expected: `docs/protocol.md: an absolute redirect is refused — exactly one request is recorded (a follow would add a second POST to the live target and succeed), the turn fails with exit 1, and the zero-auth diagnostic includes the refused Location.`,
				impact:   "Redirects are followed despite the documented refusal, so a request can silently change destination.",
				severity: "major",
			},
		}
		res := p5Run(t, spec)
		okCase := !res.timedOut && res.exit == 1 && len(res.records) == 1 &&
			res.records[0].Path == sc.Path() &&
			strings.Contains(res.stderr, sc.Redirect)
		actual := p5Actual(res) + fmt.Sprintf("; refused_location=%q records=%d", sc.Redirect, len(res.records))
		p5Conclude(t, spec, res, okCase, actual)
	})
}

// TestP5Exploratory runs three cheap undocumented probes. They have no binding
// oracle, so outcomes are observations only and never become findings.
func TestP5Exploratory(t *testing.T) {
	cases := []p5TableCase{
		{id: "X1", scenario: "p5-explore-cr-only", stdin: "hi\n", kind: "success", want: "cr only",
			oracle: "undocumented probe: lone CR line framing is not covered by docs/protocol.md, which documents LF and CRLF only."},
		{id: "X2", scenario: "p5-explore-trailing-after-done", stdin: "hi\n", kind: "success", want: "done",
			oracle: "undocumented probe: bytes after data: [DONE] are not covered by docs/protocol.md."},
		{id: "X3", scenario: "p5-explore-bom", stdin: "hi\n", kind: "success", want: "bom",
			oracle: "undocumented probe: a UTF-8 BOM before the first SSE line is not covered by docs/protocol.md."},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			spec := p5SuccessSpec(tc)
			res := p5Run(t, spec)
			observation := p5Actual(res)
			if !res.timedOut && res.exit == 0 && res.stdout == tc.want+"\n" {
				observation += "; observed behavior matches the intuitive expectation for this undocumented shape"
			}
			p5Exploratory(t, spec, res, observation)
		})
	}
}

// TestP5OptionalCut records the optional and undocumented items from
// tasks/phase-5.md that were deliberately not run, each with its reason.
func TestP5OptionalCut(t *testing.T) {
	cases := []struct{ id, item, reason string }{
		{"O1", "event larger than 1 MiB", "optional documented edge case dropped to protect the deadline (tasks/phase-5.md optional/exploratory cut)"},
		{"O2", "stream payload larger than 8 MiB", "optional documented edge case dropped to protect the deadline (tasks/phase-5.md optional/exploratory cut)"},
		{"O3", "line longer than max(context_budget, 4096)", "optional documented edge case dropped to protect the deadline (tasks/phase-5.md optional/exploratory cut)"},
		{"O4", "system prompt alone over budget", "optional documented edge case dropped to protect the deadline (tasks/phase-5.md optional/exploratory cut)"},
		{"O5", "HTTP 204 with empty body", "optional documented edge case dropped to protect the deadline (tasks/phase-5.md optional/exploratory cut)"},
		{"O6", "100 KiB non-success body snippet cap", "optional documented edge case dropped to protect the deadline (tasks/phase-5.md optional/exploratory cut)"},
		{"O7", "slow drip and idle stall (60s)", "optional slow case; the Driver instructed not to spend time on 60-second stall cases"},
		{"O8", "extra redirect codes", "optional documented edge case dropped to protect the deadline (tasks/phase-5.md optional/exploratory cut)"},
		{"O9", "byte-fragmented auth echoes", "optional documented edge case dropped to protect the deadline (tasks/phase-5.md optional/exploratory cut)"},
		{"U1", "UTF-8 BOM in the stream", "run as exploratory probe X3 instead; undocumented, no oracle"},
		{"U2", "CR-only framing", "run as exploratory probe X1 instead; undocumented, no oracle"},
		{"U3", "trailing bytes after [DONE]", "run as exploratory probe X2 instead; undocumented, no oracle"},
		{"U4", "data: spacing / multiple data fields", "undocumented probe; no oracle, dropped to protect the deadline"},
		{"U5", "Content-Type parameters / case", "undocumented probe; no oracle, dropped to protect the deadline"},
		{"U6", "invalid UTF-8 JSON", "undocumented probe; no oracle, dropped to protect the deadline"},
		{"U7", "[DONE] without prior choices", "undocumented probe; no oracle, dropped to protect the deadline"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.id, func(t *testing.T) {
			spec := p5Spec{
				id:       c.id,
				scenario: "not-applicable",
				oracle:   "none recorded; item not executed",
				notes:    c.item,
			}
			p5Skip(t, spec, c.reason+" — item: "+c.item)
		})
	}
}
