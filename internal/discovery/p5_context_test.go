//go:build mockdiscovery

package discovery

import (
	"fmt"
	"strings"
	"testing"

	"github.com/radilabs/radichat/internal/mockllm"
)

// p5DocumentedEstimate implements the documented accounting formula from
// docs/context-accounting.md: 32 units per request, 32 units per message, plus
// the UTF-8 content bytes of each message. Case budgets are derived from this
// function so a harness miscount cannot masquerade as a product divergence.
func p5DocumentedEstimate(contents ...string) int64 {
	total := int64(32) // PerRequest
	for _, c := range contents {
		total += int64(len(c)) + 32 // PerMessage
	}
	return total
}

// p5PairValid reports whether a recorded message list is a system message
// followed by zero or more complete user/assistant pairs, optionally with one
// trailing current user message. A partial pair must never be sent.
func p5PairValid(roles []string) bool {
	if len(roles) == 0 || roles[0] != "system" {
		return false
	}
	i := 1
	for i < len(roles) {
		if roles[i] != "user" {
			return false
		}
		if i+1 < len(roles) {
			if roles[i+1] != "assistant" {
				return false
			}
			i += 2
			continue
		}
		i++
	}
	return true
}

func p5RolesOf(r mockllm.Record) []string {
	roles, _ := p5RolesAndLengths(r)
	return roles
}

func p5ContentOf(r mockllm.Record, index int) (string, bool) {
	raw, ok := r.Body["messages"]
	if !ok {
		return "", false
	}
	arr, ok := raw.([]any)
	if !ok || index >= len(arr) {
		return "", false
	}
	m, ok := arr[index].(map[string]any)
	if !ok {
		return "", false
	}
	s, _ := m["content"].(string)
	return s, true
}

func p5UserContent(r mockllm.Record) string {
	roles := p5RolesOf(r)
	for i, role := range roles {
		if role == "user" {
			s, _ := p5ContentOf(r, i)
			return s
		}
	}
	return ""
}

// p5ContentsOf returns the message contents of a recorded request in order.
// Content-exact comparisons are required where shape and byte lengths cannot
// distinguish trimming the oldest pair from trimming the newest.
func p5ContentsOf(r mockllm.Record) []string {
	raw, ok := r.Body["messages"]
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
		s, _ := m["content"].(string)
		out = append(out, s)
	}
	return out
}

// TestP5Context runs C1-C11: request shape, budget boundaries, trimming,
// rollback, /clear, and the accumulated-output cap.
func TestP5Context(t *testing.T) {
	// C1 and C2: request fields and headers.
	t.Run("C1", func(t *testing.T) {
		spec := p5Spec{
			id: "C1", scenario: "context-max-tokens-sent", stdin: "hi\n",
			cfg:    p5Cfg(8192, 512),
			oracle: `docs/protocol.md#Request: body fields sent are "model", "messages", "stream" (always true), and "max_tokens" (configured generation_reserve); "Not sent: cookies, tools, functions, tool_choice, n".`,
			div: p5DivSpec{
				slug:     p5Slug("C1", "context-max-tokens-sent"),
				title:    "Case C1 (max_tokens and request shape): the request does not match the documented field set",
				expected: `docs/protocol.md#Request: the JSON body carries model, messages, stream=true, and max_tokens equal to the configured generation_reserve, with Content-Type application/json and Accept text/event-stream, and no Cookie or Authorization header in zero-auth mode.`,
				impact:   "The wire request diverges from the documented protocol subset, so endpoint behavior cannot be predicted from the documentation.",
				severity: "major",
			},
		}
		res := p5Run(t, spec)
		ok, actual := p5CheckRequestShape(res, 512)
		p5Conclude(t, spec, res, ok, actual)
	})

	t.Run("C2", func(t *testing.T) {
		spec := p5Spec{
			id: "C2", scenario: "context-no-optional-fields", stdin: "hi\n",
			cfg:    p5Cfg(8192, 768),
			oracle: `docs/protocol.md#Request: "Not sent: cookies, tools, functions, tool_choice, n, or other option bags."`,
			div: p5DivSpec{
				slug:     p5Slug("C2", "context-no-optional-fields"),
				title:    "Case C2 (no optional fields): the request carries undocumented fields",
				expected: `docs/protocol.md#Request: only model, messages, stream, and max_tokens are sent; tools, functions, tool_choice, n, cookies, and Authorization (zero-auth) are absent.`,
				impact:   "Undocumented request fields can change endpoint behavior in ways the documentation does not describe.",
				severity: "major",
			},
		}
		res := p5Run(t, spec)
		ok, actual := p5CheckRequestShape(res, 768)
		p5Conclude(t, spec, res, ok, actual)
	})

	// C3 and C4: exact budget boundary and one byte over, byte-charged.
	for _, tc := range []struct {
		id       string
		scenario string
		exact    string
		over     string
		oracle   string
	}{
		{
			id: "C3", scenario: "context-exact-budget-boundary",
			exact:  "u:" + strings.Repeat("a", 38),
			over:   "u:" + strings.Repeat("a", 39),
			oracle: `docs/context-accounting.md: "estimated prompt units plus generation_reserve must be <= context_budget"; if the protected messages cannot fit, "fail locally. No HTTP request is sent".`,
		},
		{
			id: "C4", scenario: "context-utf8-byte-accounting",
			exact:  strings.Repeat("é", 20),
			over:   strings.Repeat("é", 20) + "x",
			oracle: `docs/context-accounting.md: "charge UTF-8 content bytes of each message (len of the Go string / the raw text, not JSON-escaped size)".`,
		},
	} {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			// The budget is derived from the documented formula:
			// estimate(system, exact user) + reserve. That is exactly the
			// boundary docs/context-accounting.md defines, so the exact-fit
			// input must be sent and the one-byte-over input must fail
			// locally. Deriving it here prevents a harness miscount from
			// looking like a product divergence.
			const trimSystem = "sys-prompt"
			const trimReserve = 100
			budget := p5DocumentedEstimate(trimSystem, tc.exact) + trimReserve
			cfg := p5Cfg(budget, trimReserve).withSystem(trimSystem)

			specExact := p5Spec{
				id: tc.id, scenario: tc.scenario, stdin: tc.exact + "\n", cfg: cfg,
				notes:  "exact-fit input",
				oracle: tc.oracle,
				div: p5DivSpec{
					slug:     p5Slug(tc.id, tc.scenario+"-exact-fit"),
					title:    "Case " + tc.id + " (" + tc.scenario + ", exact fit): the boundary request is not sent",
					expected: tc.oracle + " With the documented formula the exact-fit request uses the whole budget and must still be sent (exit 0, one recorded request).",
					impact:   "A request that exactly fits the documented budget is rejected, so usable context is lost.",
					severity: "major",
				},
			}
			resExact := p5Run(t, specExact)
			okExact := resExact.exit == 0 && len(resExact.records) == 1 &&
				strings.Join(p5RolesOf(resExact.records[0]), ",") == "system,user" &&
				p5UserContent(resExact.records[0]) == tc.exact
			p5Conclude(t, specExact, resExact, okExact, p5Actual(resExact))

			specOver := p5Spec{
				id: tc.id, scenario: tc.scenario, stdin: tc.over + "\n", cfg: cfg,
				notes:  "one-byte-over input",
				oracle: tc.oracle,
				div: p5DivSpec{
					slug:     p5Slug(tc.id, tc.scenario+"-one-byte-over"),
					title:    "Case " + tc.id + " (" + tc.scenario + ", one byte over): the over-budget request is sent",
					expected: tc.oracle + " One byte over the budget must fail locally with zero HTTP requests (exit 1 in EOF mode).",
					impact:   "The client exceeds its configured working budget instead of failing locally, so a small-window model can be sent a request it cannot accept.",
					severity: "major",
				},
			}
			resOver := p5Run(t, specOver)
			okOver := resOver.exit == 1 && len(resOver.records) == 0
			p5Conclude(t, specOver, resOver, okOver, p5Actual(resOver))
		})
	}

	// C7 and C8: complete-pair trimming at budget 303 and 302.
	// system "abc" (3 bytes), 20-byte user turns, "ok" assistant replies (2
	// bytes), reserve 12: the third turn needs 291+12=303 units, so 303 keeps
	// the complete history and 302 drops the oldest complete pair.
	//
	// Dr Watson correction: roles and byte lengths alone cannot distinguish
	// dropping the oldest pair from dropping the newest (both turns are 20
	// bytes and both replies are "ok"), so the third request is asserted
	// content-exact. The trim notice is checked as nonempty stderr, because
	// the docs promise "a brief notice", not exact wording.
	trimTurn1 := "u1:" + strings.Repeat("a", 17)
	trimTurn2 := "u2:" + strings.Repeat("b", 17)
	trimTurn3 := "u3:" + strings.Repeat("c", 17)
	trimStdin := trimTurn1 + "\n" + trimTurn2 + "\n" + trimTurn3 + "\n"
	const trimAssistant = "ok" // the context-pair-trimming scenario's reply
	for _, tc := range []struct {
		id         string
		budget     int64
		wantLast   []string
		wantNotice bool
	}{
		{id: "C7", budget: 303, wantNotice: false,
			wantLast: []string{"abc", trimTurn1, trimAssistant, trimTurn2, trimAssistant, trimTurn3}},
		{id: "C8", budget: 302, wantNotice: true,
			wantLast: []string{"abc", trimTurn2, trimAssistant, trimTurn3}},
	} {
		tc := tc
		t.Run(tc.id, func(t *testing.T) {
			spec := p5Spec{
				id: tc.id, scenario: "context-pair-trimming", stdin: trimStdin,
				cfg:    p5Cfg(tc.budget, 12).withSystem("abc"),
				oracle: `docs/context-accounting.md#Trimming: "drop the oldest complete user/assistant pairs until the request fits", "Print a brief notice on stderr when pairs are dropped", and "The system message and the current user message are never trimmed".`,
				div: p5DivSpec{
					slug:     p5Slug(tc.id, "context-pair-trimming"),
					title:    "Case " + tc.id + " (pair trimming at budget " + fmt.Sprint(tc.budget) + "): trimming does not follow the documented policy",
					expected: `docs/context-accounting.md: at budget 303 the third request retains the complete history with contents [abc, u1-turn, ok, u2-turn, ok, u3-turn]; at 302 it drops exactly the oldest complete pair, so the third request contents are [abc, u2-turn, ok, u3-turn], never a partial pair, and the trimmed case emits a brief nonempty stderr notice while the untrimmed case emits none.`,
					impact:   "History trimming diverges from the documented deterministic policy, so the effective context seen by the model differs from what the documentation promises.",
					severity: "major",
				},
			}
			res := p5Run(t, spec)
			ok := res.exit == 0 && len(res.records) == 3
			actual := p5Actual(res)
			if ok {
				last := p5ContentsOf(res.records[2])
				if strings.Join(last, "|") != strings.Join(tc.wantLast, "|") {
					ok = false
					actual += "; third request contents=[" + strings.Join(last, "|") + "]"
				}
				for i, r := range res.records {
					if !p5PairValid(p5RolesOf(r)) {
						ok = false
						actual += fmt.Sprintf("; request %d sent a partial pair: %v", i+1, p5RolesOf(r))
					}
				}
				// The docs promise a brief notice when pairs are dropped, not
				// exact wording: the trimmed case must emit some nonempty
				// stderr notice, and the untrimmed case must emit none.
				if (res.stderr != "") != tc.wantNotice {
					ok = false
					actual += fmt.Sprintf("; notice present=%v want=%v", res.stderr != "", tc.wantNotice)
				}
			} else {
				actual += fmt.Sprintf("; recorded roles per request: %v", p5AllRoles(res))
			}
			p5Conclude(t, spec, res, ok, actual)
		})
	}

	// C9: failed-turn rollback across two piped input lines.
	t.Run("C9", func(t *testing.T) {
		spec := p5Spec{
			id: "C9", scenario: "context-failed-turn-rollback", stdin: "first-line\nsecond-line\n",
			cfg:    p5Cfg(8192, 1024).withSystem("sys"),
			oracle: `docs/context-accounting.md#Commit-and-rollback: "The trimmed history, new user message, and assistant reply are committed only after a stream completes successfully", and protocol errors "abort the staged turn and restore the previous committed conversation".`,
			div: p5DivSpec{
				slug:     p5Slug("C9", "context-failed-turn-rollback"),
				title:    "Case C9 (failed-turn rollback): a failed turn leaks into the next request",
				expected: `docs/context-accounting.md: after a failed stream the staged turn is discarded, so the next request carries only the previously committed context plus the new user message — never the failed turn's user text or partial assistant text.`,
				impact:   "A failed turn is committed as if it succeeded, corrupting subsequent conversation context.",
				severity: "major",
			},
		}
		res := p5Run(t, spec)
		ok := res.exit == 1 && len(res.records) == 2
		actual := p5Actual(res)
		if ok {
			second := p5RolesOf(res.records[1])
			if strings.Join(second, ",") != "system,user" {
				ok = false
				actual += "; second request roles=" + strings.Join(second, ",")
			}
			if got := p5UserContent(res.records[1]); got != "second-line" {
				ok = false
				actual += "; second request user content=" + fmt.Sprintf("%q", got)
			}
			for i, r := range res.records {
				if strings.Contains(p5UserContent(r), "partial") {
					ok = false
					actual += fmt.Sprintf("; request %d carries failed-turn text", i+1)
				}
			}
			if !strings.Contains(res.stdout, "partial") {
				ok = false
				actual += "; partial stream output was not printed before the failure"
			}
		}
		p5Conclude(t, spec, res, ok, actual)
	})

	// C10: /clear resets history and keeps the system prompt.
	t.Run("C10", func(t *testing.T) {
		spec := p5Spec{
			id: "C10", scenario: "context-after-clear", stdin: "one\n/clear\ntwo\n",
			cfg:    p5Cfg(8192, 1024).withSystem("sys"),
			oracle: `docs/context-accounting.md: "/clear drops committed and staged turns and keeps the configured system prompt."`,
			div: p5DivSpec{
				slug:     p5Slug("C10", "context-after-clear"),
				title:    "Case C10 (after /clear): history is not reset as documented",
				expected: `docs/context-accounting.md: after /clear the next request carries only the system message and the new user message.`,
				impact:   "The /clear command does not reset conversation state, so the user cannot reclaim context.",
				severity: "major",
			},
		}
		res := p5Run(t, spec)
		ok := res.exit == 0 && len(res.records) == 2
		actual := p5Actual(res)
		if ok {
			for i, want := range []string{"one", "two"} {
				rec := res.records[i]
				if strings.Join(p5RolesOf(rec), ",") != "system,user" || p5UserContent(rec) != want {
					ok = false
					actual += fmt.Sprintf("; request %d roles=%v user=%q want %q", i+1, p5RolesOf(rec), p5UserContent(rec), want)
				}
			}
		}
		p5Conclude(t, spec, res, ok, actual)
	})

	// C11: accumulated assistant output at and one byte over context_budget.
	for _, tc := range []struct {
		id       string
		scenario string
		wantLen  int
		success  bool
	}{
		{id: "C11", scenario: "p5-output-exact", wantLen: p5OutputBytes, success: true},
		{id: "C11", scenario: "p5-output-over", wantLen: 0, success: false},
	} {
		tc := tc
		t.Run(tc.id+"/"+tc.scenario, func(t *testing.T) {
			notes := "exact cap"
			if !tc.success {
				notes = "one byte over the cap"
			}
			spec := p5Spec{
				id: tc.id, scenario: tc.scenario, stdin: "go\n",
				cfg:    p5Cfg(p5OutputBytes, 1024),
				notes:  notes,
				oracle: `docs/protocol.md#Timeouts-and-limits: "Accumulated assistant UTF-8 bytes | context_budget (exceeding is an error; the turn is discarded, never silently truncated)".`,
				div: p5DivSpec{
					slug:     p5Slug(tc.id, tc.scenario),
					title:    "Case C11 (" + notes + "): the accumulated-output bound does not follow the documented limit",
					expected: `docs/protocol.md: accumulated assistant output of exactly context_budget bytes completes (exit 0), and one byte over fails with the turn discarded and not committed (exit 1), never silently truncated.`,
					impact:   "The local output bound diverges from the documented context_budget limit, so oversized completions are either rejected early or accepted past the budget.",
					severity: "major",
				},
			}
			res := p5Run(t, spec)
			var ok bool
			if tc.success {
				ok = res.exit == 0 && len(res.stdout) == tc.wantLen+1 && len(res.records) == 1
			} else {
				ok = res.exit == 1 && len(res.records) == 1
			}
			actual := p5Actual(res) + fmt.Sprintf("; stdout_bytes=%d", len(res.stdout))
			p5Conclude(t, spec, res, ok, actual)
		})
	}
}

// p5CheckRequestShape verifies the documented request field set and headers
// for the C1/C2 fixtures.
func p5CheckRequestShape(res *p5Result, wantMaxTokens int) (bool, string) {
	ok := res.exit == 0 && len(res.records) == 1
	actual := p5Actual(res)
	if !ok {
		return ok, actual
	}
	rec := res.records[0]
	if len(rec.Body) != 4 {
		ok = false
		actual += fmt.Sprintf("; body has %d fields, want exactly 4", len(rec.Body))
	}
	for _, k := range []string{"model", "messages", "stream", "max_tokens"} {
		if _, present := rec.Body[k]; !present {
			ok = false
			actual += "; missing body field " + k
		}
	}
	for _, k := range []string{"tools", "functions", "tool_choice", "n"} {
		if _, present := rec.Body[k]; present {
			ok = false
			actual += "; undocumented body field present: " + k
		}
	}
	if v, present := rec.Body["stream"]; !present || v != true {
		ok = false
		actual += fmt.Sprintf("; stream=%v want true", v)
	}
	if v, present := rec.Body["model"]; !present || v != "p5-model" {
		ok = false
		actual += fmt.Sprintf("; model=%v want p5-model", v)
	}
	if v, present := rec.Body["max_tokens"]; !present || v != float64(wantMaxTokens) {
		ok = false
		actual += fmt.Sprintf("; max_tokens=%v want %d", v, wantMaxTokens)
	}
	if rec.Headers["Content-Type"] != "application/json" {
		ok = false
		actual += "; Content-Type=" + rec.Headers["Content-Type"]
	}
	if rec.Headers["Accept"] != "text/event-stream" {
		ok = false
		actual += "; Accept=" + rec.Headers["Accept"]
	}
	if _, present := rec.Headers["Cookie"]; present {
		ok = false
		actual += "; Cookie header present"
	}
	if _, present := rec.Headers["Authorization"]; present {
		ok = false
		actual += "; Authorization header present in zero-auth mode"
	}
	return ok, actual
}

func p5AllRoles(res *p5Result) []string {
	out := make([]string, 0, len(res.records))
	for _, r := range res.records {
		out = append(out, strings.Join(p5RolesOf(r), ","))
	}
	return out
}
