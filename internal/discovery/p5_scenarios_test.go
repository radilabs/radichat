//go:build mockdiscovery

package discovery

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/radilabs/radichat/internal/mockllm"
)

// p5OutputBytes is the accumulated-assistant-output boundary exercised by case
// C11. The scenario payloads are static, so the C11 config must use this value
// as its context_budget for the exact-fit and one-byte-over inputs.
const p5OutputBytes = 8192

func init() {
	registerP5EchoScenarios()
	registerP5HTTPErrorScenarios()
	registerP5OutputCapScenarios()
	registerP5SubstituteScenarios()
	registerP5ExploratoryScenarios()
}

// p5Chunk builds one SSE data event carrying a content delta and an optional
// finish reason. It mirrors the accepted catalog's event shape; the mock's own
// builders are unexported, so the harness builds its own bytes here.
func p5Chunk(content, finish string) string {
	delta := map[string]any{"delta": map[string]any{}}
	if content != "" {
		delta["delta"].(map[string]any)["content"] = content
	}
	if finish != "" {
		delta["finish_reason"] = finish
	} else {
		delta["finish_reason"] = nil
	}
	raw, err := json.Marshal(map[string]any{"choices": []any{delta}})
	if err != nil {
		panic("discovery: cannot build chunk: " + err.Error())
	}
	return "data: " + string(raw) + "\n\n"
}

func p5RawEvent(payload string) string { return "data: " + payload + "\n\n" }

const p5Done = "data: [DONE]\n\n"

func p5SSEHeaders() map[string]string {
	return map[string]string{"Content-Type": "text/event-stream"}
}

// registerP5EchoScenarios registers the token-echo scenarios used by cases
// E1-E3 and E7. They deliberately do not require authentication so the same
// scenario can also demonstrate that zero-auth stream text is unchanged.
func registerP5EchoScenarios() {
	mockllm.Register(mockllm.Scenario{
		Name:        "p5-echo-delta",
		Family:      "p5",
		Description: "Harness-local: one delta whose content embeds the sentinel credential between ordinary text.",
		Headers:     p5SSEHeaders(),
		Wire:        mockllm.FragmentText(p5Chunk("before "+mockllm.SentinelToken+" after", "stop") + p5Done),
		Mode:        mockllm.ModeHTTP,
	})
	mockllm.Register(mockllm.Scenario{
		Name:        "p5-echo-split",
		Family:      "p5",
		Description: "Harness-local: the sentinel credential split across two content deltas.",
		Headers:     p5SSEHeaders(),
		Wire: mockllm.FragmentText(
			p5Chunk("before "+mockllm.SentinelToken[:20], "") +
				p5Chunk(mockllm.SentinelToken[20:]+" after", "stop") + p5Done),
		Mode: mockllm.ModeHTTP,
	})
	mockllm.Register(mockllm.Scenario{
		Name:        "p5-echo-trailing-prefix",
		Family:      "p5",
		Description: "Harness-local: content ends on a proper prefix of the sentinel credential.",
		Headers:     p5SSEHeaders(),
		Wire:        mockllm.FragmentText(p5Chunk("keep this "+mockllm.SentinelToken[:8], "stop") + p5Done),
		Mode:        mockllm.ModeHTTP,
	})
}

// registerP5HTTPErrorScenarios registers the diagnostic-echo scenarios used by
// cases E4-E6: untrusted bodies, stream error payloads, and redirect locations
// that embed the sentinel credential and must not be printed in authenticated
// mode.
func registerP5HTTPErrorScenarios() {
	mockllm.Register(mockllm.Scenario{
		Name:        "p5-http-401-echo",
		Family:      "p5",
		Description: "Harness-local: HTTP 401 whose body embeds the sentinel credential.",
		Status:      401,
		Headers:     map[string]string{"Content-Type": "text/plain"},
		Wire:        mockllm.FragmentText("unauthorized: " + mockllm.SentinelToken + "\n"),
		Mode:        mockllm.ModeHTTP,
	})
	mockllm.Register(mockllm.Scenario{
		Name:        "p5-http-500-echo",
		Family:      "p5",
		Description: "Harness-local: HTTP 500 whose body embeds the sentinel credential.",
		Status:      500,
		Headers:     map[string]string{"Content-Type": "text/html"},
		Wire:        mockllm.FragmentText("<html><body>boom " + mockllm.SentinelToken + "</body></html>"),
		Mode:        mockllm.ModeHTTP,
	})
	mockllm.Register(mockllm.Scenario{
		Name:        "p5-stream-error-echo",
		Family:      "p5",
		Description: "Harness-local: stream error object whose message embeds the sentinel credential.",
		Headers:     p5SSEHeaders(),
		Wire: mockllm.FragmentText(
			p5RawEvent(`{"error":{"message":"boom `+mockllm.SentinelToken+`","type":"server_error"}}`) + p5Done),
		Mode: mockllm.ModeHTTP,
	})
	mockllm.Register(mockllm.Scenario{
		Name:        "p5-redirect-echo",
		Family:      "p5",
		Description: "Harness-local: redirect whose Location embeds the sentinel credential.",
		Redirect:    "/v1/chat/completions/stream-success-basic?note=" + mockllm.SentinelToken,
		Headers:     p5SSEHeaders(),
		Mode:        mockllm.ModeHTTP,
	})
}

// registerP5OutputCapScenarios registers the accumulated-output boundary
// scenarios used by case C11.
func registerP5OutputCapScenarios() {
	mockllm.Register(mockllm.Scenario{
		Name:        "p5-output-exact",
		Family:      "p5",
		Description: "Harness-local: successful stream whose accumulated assistant text is exactly p5OutputBytes bytes.",
		Headers:     p5SSEHeaders(),
		Wire:        mockllm.FragmentText(p5Chunk(strings.Repeat("x", p5OutputBytes), "stop") + p5Done),
		Mode:        mockllm.ModeHTTP,
	})
	mockllm.Register(mockllm.Scenario{
		Name:        "p5-output-over",
		Family:      "p5",
		Description: "Harness-local: successful stream whose accumulated assistant text is one byte over p5OutputBytes.",
		Headers:     p5SSEHeaders(),
		Wire:        mockllm.FragmentText(p5Chunk(strings.Repeat("x", p5OutputBytes+1), "stop") + p5Done),
		Mode:        mockllm.ModeHTTP,
	})
}

// p5RedirectAbsoluteLiveOnce guards runtime registration of the live
// absolute-redirect scenario. Registration cannot happen in init() because the
// mock server — and therefore its URL — only exists once TestMain has started
// it, and the Location must point at that live server.
var p5RedirectAbsoluteLiveOnce sync.Once

// registerP5RedirectAbsoluteLive registers a harness-local scenario whose
// absolute Location points at this same live mock server's
// stream-success-basic route. If RadiChat followed the redirect, the same
// recorder would show a second POST and a success, which makes refusal
// directly observable — unlike the accepted redirect-absolute scenario, whose
// 127.0.0.1:1 target cannot distinguish a refusal from a dial failure.
func registerP5RedirectAbsoluteLive() {
	p5RedirectAbsoluteLiveOnce.Do(func() {
		mockllm.Register(mockllm.Scenario{
			Name:        "p5-redirect-absolute-live",
			Family:      "p5",
			Description: "Harness-local: 302 redirect whose absolute Location points at this same mock server's stream-success-basic route, so a followed redirect would produce a second recorded request and a success.",
			Redirect:    p5Server.URL() + "/v1/chat/completions/stream-success-basic",
			Headers:     p5SSEHeaders(),
			Mode:        mockllm.ModeHTTP,
		})
	})
}

// registerP5SubstituteScenarios registers payloads the accepted Phase 4
// catalog cannot deliver. The accepted stream-success-no-content-type
// scenario declares no Content-Type, but the mock's HTTP mode writes the body
// through net/http, which sniffs and sets "text/plain; charset=utf-8" (verified
// with curl during Phase 5). Raw mode writes the configured headers verbatim,
// so this payload truly omits the header and exercises the documented branch.
func registerP5SubstituteScenarios() {
	mockllm.Register(mockllm.Scenario{
		Name:        "p5-no-content-type-raw",
		Family:      "p5",
		Description: "Harness-local: successful stream with the Content-Type header truly omitted (raw mode; net/http would otherwise sniff one in).",
		Headers:     map[string]string{},
		Wire:        mockllm.FragmentText(p5Chunk("untyped", "stop") + p5Done),
		Mode:        mockllm.ModeRaw,
	})
}

// registerP5ExploratoryScenarios registers undocumented probes. Their outcomes
// are exploratory observations only: no documented oracle covers them, so they
// never become findings.
func registerP5ExploratoryScenarios() {
	lf := p5Chunk("cr only", "stop") + p5Done
	mockllm.Register(mockllm.Scenario{
		Name:        "p5-explore-cr-only",
		Family:      "p5",
		Description: "Undocumented probe: stream framed with lone CR line endings.",
		Headers:     p5SSEHeaders(),
		Wire:        mockllm.FragmentText(strings.ReplaceAll(lf, "\n", "\r")),
		Mode:        mockllm.ModeHTTP,
	})
	mockllm.Register(mockllm.Scenario{
		Name:        "p5-explore-trailing-after-done",
		Family:      "p5",
		Description: "Undocumented probe: extra event bytes after data: [DONE].",
		Headers:     p5SSEHeaders(),
		Wire:        mockllm.FragmentText(p5Chunk("done", "stop") + p5Done + p5Chunk("late", "")),
		Mode:        mockllm.ModeHTTP,
	})
	mockllm.Register(mockllm.Scenario{
		Name:        "p5-explore-bom",
		Family:      "p5",
		Description: "Undocumented probe: UTF-8 BOM before the first SSE line.",
		Headers:     p5SSEHeaders(),
		Wire:        mockllm.FragmentText("\ufeff" + p5Chunk("bom", "stop") + p5Done),
		Mode:        mockllm.ModeHTTP,
	})
}
