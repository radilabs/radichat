package mockllm

import (
	"encoding/json"
	"net/http"
)

// This file declares the scenario catalog. Each scenario is data: the exact
// bytes it writes, the status and headers it sets, and the request shape it
// observes. Adding a scenario never requires changing the server.

// Sentinel credentials used by the mock's own tests and by the documented
// examples. They are obviously fake values, never real credentials.
const (
	// SentinelToken is the expected bearer credential for authenticated
	// scenarios.
	SentinelToken = "mockllm-sentinel-token-not-a-real-credential"

	// SentinelMismatchToken is a credential that does not match
	// SentinelToken, for exercising the mismatch path.
	SentinelMismatchToken = "mockllm-mismatch-token-not-a-real-credential"

	// SentinelEnvName is the environment variable name that
	// docs/testing-mock.md tells operators to export the sentinel token in.
	// The name is safe to print; the value is not.
	SentinelEnvName = "MOCKLLM_SENTINEL_TOKEN"
)

// SSE content type and the terminal marker RadiChat requires.
const (
	contentTypeSSE  = "text/event-stream"
	contentTypeJSON = "application/json"
	doneMarker      = "data: [DONE]\n\n"
)

// chunk builds one SSE data event carrying a content delta and an optional
// finish reason.
func chunk(content, finish string) string {
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
		panic("mockllm: cannot build chunk: " + err.Error())
	}
	return "data: " + string(raw) + "\n\n"
}

// rawEvent builds one SSE data event from an arbitrary payload string.
func rawEvent(payload string) string { return "data: " + payload + "\n\n" }

// successWire builds the canonical successful stream: one content delta, one
// terminal delta carrying the finish reason, then [DONE].
func successWire(content, finish string) []Segment {
	wire := make([]byte, 0, 128)
	wire = append(wire, chunk(content, "")...)
	wire = append(wire, chunk("", finish)...)
	wire = append(wire, doneMarker...)
	return FragmentAt(wire)
}

// successWireMulti builds a stream whose content is delivered across several
// deltas.
func successWireMulti(parts ...string) []Segment {
	wire := make([]byte, 0, 128)
	for i, p := range parts {
		finish := ""
		if i == len(parts)-1 {
			finish = "stop"
		}
		wire = append(wire, chunk(p, finish)...)
	}
	wire = append(wire, doneMarker...)
	return FragmentAt(wire)
}

// concat joins strings and segments into one segment slice.
func concat(parts ...any) []Segment {
	wire := make([]byte, 0, 256)
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			wire = append(wire, v...)
		case Segment:
			wire = append(wire, v.Bytes...)
		case []Segment:
			for _, s := range v {
				wire = append(wire, s.Bytes...)
			}
		}
	}
	return FragmentAt(wire)
}

// crlf converts LF line endings to CRLF.
func crlf(s string) string {
	out := make([]byte, 0, len(s)+8)
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if len(out) == 0 || out[len(out)-1] != '\r' {
				out = append(out, '\r')
			}
		}
		out = append(out, s[i])
	}
	return string(out)
}

// FragmentEveryN splits data into segments of at most n bytes each.
func FragmentEveryN(data []byte, n int) []Segment {
	if n <= 0 {
		return nil
	}
	out := make([]Segment, 0, len(data)/n+1)
	for i := 0; i < len(data); i += n {
		end := i + n
		if end > len(data) {
			end = len(data)
		}
		out = append(out, Segment{Bytes: data[i:end]})
	}
	return out
}

func init() {
	registerStreamScenarios()
	registerFramingScenarios()
	registerMalformedScenarios()
	registerHTTPErrorScenarios()
	registerAuthScenarios()
	registerRedirectScenarios()
	registerContextScenarios()
}

// sseHeaders are the headers every successful streaming scenario sets.
var sseHeaders = map[string]string{"Content-Type": contentTypeSSE}

func registerStreamScenarios() {
	Register(Scenario{
		Name:        "stream-success-basic",
		Family:      "stream",
		Description: "Successful streamed response: one content delta, a stop finish reason, then [DONE].",
		Headers:     sseHeaders,
		Wire:        successWire("hello", "stop"),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Fields: map[string]any{"stream": true},
			Note:   "Baseline successful exchange; RadiChat prints the assistant text and commits the turn.",
		},
	})

	Register(Scenario{
		Name:        "stream-success-multibyte",
		Family:      "stream",
		Description: "Successful stream whose content is multibyte UTF-8 (accents, emoji, CJK) split across deltas.",
		Headers:     sseHeaders,
		Wire:        successWireMulti("héllo", "🌍", "日本語"),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "UTF-8 content must be reassembled byte for byte; a rune must not become a replacement character.",
		},
	})

	Register(Scenario{
		Name:        "stream-success-length-finish",
		Family:      "stream",
		Description: "Successful stream that ends with finish_reason length, which RadiChat treats as success.",
		Headers:     sseHeaders,
		Wire:        successWire("truncated text", "length"),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "A length finish is a success: RadiChat commits the complete streamed text without local truncation.",
		},
	})

	Register(Scenario{
		Name:        "stream-success-role-only",
		Family:      "stream",
		Description: "Stream whose first delta carries only a role, then empty deltas, then content.",
		Headers:     sseHeaders,
		Wire: concat(
			rawEvent(`{"choices":[{"delta":{"role":"assistant"},"finish_reason":null}]}`),
			rawEvent(`{"choices":[{"delta":{},"finish_reason":null}]}`),
			chunk("ok", "stop"),
			doneMarker,
		),
		Mode: ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "Role-only chunks and empty deltas are legal and contribute no assistant text.",
		},
	})

	Register(Scenario{
		Name:        "stream-success-crlf",
		Family:      "stream",
		Description: "Successful stream framed with CRLF line endings.",
		Headers:     sseHeaders,
		Wire:        FragmentText(string(crlf(chunk("crlf text", "stop") + doneMarker))),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "LF and CRLF framing are both accepted.",
		},
	})

	Register(Scenario{
		Name:        "stream-success-comments",
		Family:      "stream",
		Description: "Successful stream with SSE comments and ignored event, id, and retry fields.",
		Headers:     sseHeaders,
		Wire: concat(
			": keep-alive comment\r\n",
			"event: message\r\nid: 1\r\nretry: 100\r\n",
			chunk("commented", ""),
			"\n",
			chunk("", "stop"),
			doneMarker,
		),
		Mode: ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "Comment lines and non-data fields are ignored; only data payloads are consumed.",
		},
	})

	Register(Scenario{
		Name:        "stream-success-no-content-type",
		Family:      "stream",
		Description: "Successful stream with the Content-Type header omitted, which RadiChat still parses as SSE.",
		Headers:     map[string]string{},
		Wire:        successWire("untyped", "stop"),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "An omitted Content-Type is parsed as SSE; a present but wrong one is not.",
		},
	})

	Register(Scenario{
		Name:        "stream-success-empty-content",
		Family:      "stream",
		Description: "Stream that completes with no assistant text at all.",
		Headers:     sseHeaders,
		Wire:        FragmentText(chunk("", "stop") + doneMarker),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "A choice with no content is a valid empty completion, not an error.",
		},
	})
}

func registerFramingScenarios() {
	// A single successful stream, then fragmented at offsets chosen to land
	// inside a multibyte rune, inside a CRLF, inside a data: field, and
	// inside the [DONE] marker.
	multi := chunk("héllo", "stop") + doneMarker

	Register(Scenario{
		Name:        "fragment-utf8-inside-rune",
		Family:      "framing",
		Description: "One successful stream fragmented at a byte offset inside a multibyte UTF-8 sequence.",
		Headers:     sseHeaders,
		Wire:        Fragment([]byte(multi), midRuneOffset(multi)),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "A rune split across writes must be reassembled, not replaced with U+FFFD.",
		},
	})

	crlfWire := crlf(chunk("crlf", "stop") + doneMarker)
	Register(Scenario{
		Name:        "fragment-inside-crlf",
		Family:      "framing",
		Description: "One successful CRLF-framed stream fragmented between the CR and LF of a line ending.",
		Headers:     sseHeaders,
		Wire:        Fragment([]byte(crlfWire), firstCRLFSplit(crlfWire)),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "A CRLF split across writes must not be mistaken for two line endings.",
		},
	})

	dataWire := chunk("frag", "stop") + doneMarker
	Register(Scenario{
		Name:        "fragment-inside-data-field",
		Family:      "framing",
		Description: "One successful stream fragmented inside a data: field, between the field name and its payload.",
		Headers:     sseHeaders,
		Wire:        Fragment([]byte(dataWire), firstDataFieldSplit(dataWire)),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "A data: field split across writes must be reassembled before parsing.",
		},
	})

	doneWire := chunk("done", "stop") + doneMarker
	Register(Scenario{
		Name:        "fragment-inside-done",
		Family:      "framing",
		Description: "One successful stream fragmented inside the [DONE] marker.",
		Headers:     sseHeaders,
		Wire:        Fragment([]byte(doneWire), firstDoneSplit(doneWire)),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "The [DONE] marker must be recognized even when split across writes.",
		},
	})

	Register(Scenario{
		Name:        "fragment-byte-by-byte",
		Family:      "framing",
		Description: "One successful stream delivered one byte at a time.",
		Headers:     sseHeaders,
		Wire:        FragmentEveryN([]byte(multi), 1),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "Extreme fragmentation still yields the same assembled assistant text.",
		},
	})

	Register(Scenario{
		Name:        "fragment-multiple-events-one-write",
		Family:      "framing",
		Description: "Several complete events delivered in a single write, the opposite of fragmentation.",
		Headers:     sseHeaders,
		Wire: concat(
			chunk("one", ""),
			chunk("two", ""),
			chunk("", "stop"),
			doneMarker,
		),
		Mode: ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "Multiple events in one read must all be processed.",
		},
	})
}

func registerMalformedScenarios() {
	Register(Scenario{
		Name:        "malformed-json-event",
		Family:      "malformed",
		Description: "Stream containing a data event whose payload is not valid JSON.",
		Headers:     sseHeaders,
		Wire:        FragmentText(rawEvent("{not json") + doneMarker),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "Malformed stream JSON fails with a clear parse error."},
	})

	Register(Scenario{
		Name:        "malformed-delta-content-not-string",
		Family:      "malformed",
		Description: "Stream whose delta.content is a number rather than a JSON string.",
		Headers:     sseHeaders,
		Wire:        FragmentText(rawEvent(`{"choices":[{"delta":{"content":42},"finish_reason":null}]}`) + doneMarker),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "Non-string delta content fails as an unsupported protocol subset."},
	})

	Register(Scenario{
		Name:        "malformed-multiple-choices",
		Family:      "malformed",
		Description: "Stream event carrying more than one choice.",
		Headers:     sseHeaders,
		Wire: FragmentText(rawEvent(`{"choices":[{"delta":{"content":"a"},"finish_reason":null},`+
			`{"delta":{"content":"b"},"finish_reason":null}]}`) + doneMarker),
		Mode:   ModeHTTP,
		Expect: Expect{Method: "POST", Note: "Multiple choices fail; RadiChat supports exactly one."},
	})

	Register(Scenario{
		Name:        "malformed-early-termination",
		Family:      "malformed",
		Description: "Stream that ends before [DONE], with no terminal marker at all.",
		Headers:     sseHeaders,
		Wire: []Segment{
			{Bytes: []byte(chunk("partial", ""))},
			{Bytes: []byte{}, CloseAfter: true},
		},
		Mode:   ModeHTTP,
		Expect: Expect{Method: "POST", Note: "Premature EOF without [DONE] fails and the turn is not committed."},
	})

	Register(Scenario{
		Name:        "malformed-early-termination-mid-event",
		Family:      "malformed",
		Description: "Stream truncated in the middle of an event, after partial JSON has been written.",
		Headers:     sseHeaders,
		Wire: []Segment{
			{Bytes: []byte(chunk("partial", ""))},
			{Bytes: []byte(`data: {"choices":[{"delta"`), CloseAfter: true},
		},
		Mode:   ModeHTTP,
		Expect: Expect{Method: "POST", Note: "Truncation inside an event also fails; partial data is never committed."},
	})

	Register(Scenario{
		Name:        "stream-error-object",
		Family:      "malformed",
		Description: "Stream carrying a JSON error object.",
		Headers:     sseHeaders,
		Wire:        FragmentText(rawEvent(`{"error":{"message":"model unavailable","type":"server_error"}}`) + doneMarker),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "A stream error object fails as an endpoint error."},
	})

	Register(Scenario{
		Name:        "stream-error-non-object",
		Family:      "malformed",
		Description: "Stream carrying a non-null error value that is not a JSON object.",
		Headers:     sseHeaders,
		Wire:        FragmentText(rawEvent(`{"error":["boom"]}`) + doneMarker),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "A malformed error payload fails closed."},
	})

	Register(Scenario{
		Name:        "stream-error-null-ignored",
		Family:      "malformed",
		Description: "Stream carrying an explicit null error, which is not an error.",
		Headers:     sseHeaders,
		Wire:        FragmentText(rawEvent(`{"error":null,"choices":[{"delta":{"content":"fine"},"finish_reason":"stop"}]}`) + doneMarker),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "JSON null error and an omitted error field are both ignored."},
	})

	Register(Scenario{
		Name:        "stream-tool-calls-rejected",
		Family:      "malformed",
		Description: "Stream delta carrying tool_calls.",
		Headers:     sseHeaders,
		Wire:        FragmentText(rawEvent(`{"choices":[{"delta":{"tool_calls":[{"id":"c1","function":{"name":"f","arguments":"{}"}}]},"finish_reason":null}]}`) + doneMarker),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "Tool-call deltas fail; tools are outside the supported subset."},
	})

	Register(Scenario{
		Name:        "stream-function-call-rejected",
		Family:      "malformed",
		Description: "Stream delta carrying function_call.",
		Headers:     sseHeaders,
		Wire:        FragmentText(rawEvent(`{"choices":[{"delta":{"function_call":{"name":"f","arguments":"{}"}},"finish_reason":null}]}`) + doneMarker),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "Function-call deltas fail; functions are outside the supported subset."},
	})

	Register(Scenario{
		Name:        "stream-finish-content-filter",
		Family:      "malformed",
		Description: "Stream that completes with finish_reason content_filter.",
		Headers:     sseHeaders,
		Wire:        FragmentText(chunk("", "content_filter") + doneMarker),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "content_filter fails with a clear refusal message."},
	})

	Register(Scenario{
		Name:        "stream-finish-unsupported",
		Family:      "malformed",
		Description: "Stream that completes with an unrecognized nonempty finish_reason.",
		Headers:     sseHeaders,
		Wire:        FragmentText(chunk("", "weird_reason") + doneMarker),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "An unsupported finish reason fails rather than being ignored."},
	})
}

func registerHTTPErrorScenarios() {
	Register(Scenario{
		Name:        "http-error-401",
		Family:      "http-error",
		Description: "HTTP 401 with a text/plain body.",
		Status:      http.StatusUnauthorized,
		Headers:     map[string]string{"Content-Type": "text/plain"},
		Wire:        FragmentText(string("unauthorized\n")),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "A 401 status fails with the status code and a fix hint."},
	})

	Register(Scenario{
		Name:        "http-error-429",
		Family:      "http-error",
		Description: "HTTP 429 with a JSON error body.",
		Status:      http.StatusTooManyRequests,
		Headers:     map[string]string{"Content-Type": contentTypeJSON},
		Wire:        FragmentText(string(`{"error":{"message":"rate limited"}}`)),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "A 429 status fails without any retry."},
	})

	Register(Scenario{
		Name:        "http-error-500",
		Family:      "http-error",
		Description: "HTTP 500 with an HTML error body.",
		Status:      http.StatusInternalServerError,
		Headers:     map[string]string{"Content-Type": "text/html"},
		Wire:        FragmentText(string("<html><body>internal error</body></html>")),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "A 500 status fails with the status code and a fix hint."},
	})

	Register(Scenario{
		Name:        "http-error-404-unknown-model",
		Family:      "http-error",
		Description: "HTTP 404 with a JSON body naming an unknown model.",
		Status:      http.StatusNotFound,
		Headers:     map[string]string{"Content-Type": contentTypeJSON},
		Wire:        FragmentText(string(`{"error":{"message":"model not found","code":"model_not_found"}}`)),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "A 404 for a bad model name fails with the status and a fix hint."},
	})

	Register(Scenario{
		Name:        "non-sse-content-type",
		Family:      "http-error",
		Description: "HTTP 200 with a non-SSE Content-Type and a non-streaming JSON body.",
		Headers:     map[string]string{"Content-Type": contentTypeJSON},
		Wire:        FragmentText(string(`{"choices":[{"message":{"role":"assistant","content":"non-streaming"}}]}`)),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "A non-streaming JSON response fails; there is no fallback."},
	})

	Register(Scenario{
		Name:        "non-sse-empty-body",
		Family:      "http-error",
		Description: "HTTP 200 with no Content-Type and no body at all.",
		Headers:     map[string]string{},
		Wire:        nil,
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "An empty body with no content type ends before [DONE] and fails."},
	})
}

func registerAuthScenarios() {
	Register(Scenario{
		Name:        "auth-required-match",
		Family:      "auth",
		Description: "Scenario that requires a bearer credential matching the expected token.",
		Auth:        AuthSpec{Required: true, Token: SentinelToken},
		Headers:     sseHeaders,
		Wire:        successWire("authorized", "stop"),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "With the expected credential the stream succeeds exactly as the zero-auth path does."},
	})

	Register(Scenario{
		Name:        "auth-required-missing",
		Family:      "auth",
		Description: "Scenario that requires a bearer credential, exercised without one.",
		Auth:        AuthSpec{Required: true, Token: SentinelToken},
		Headers:     sseHeaders,
		Wire:        successWire("never reached", "stop"),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "Without a credential the server answers 401 and no stream is produced."},
	})

	Register(Scenario{
		Name:        "auth-required-mismatch",
		Family:      "auth",
		Description: "Scenario that requires a bearer credential, exercised with a different token.",
		Auth:        AuthSpec{Required: true, Token: SentinelToken},
		Headers:     sseHeaders,
		Wire:        successWire("never reached", "stop"),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "A wrong credential answers 401; the credential value is never recorded or echoed."},
	})

	Register(Scenario{
		Name:        "auth-malformed-header",
		Family:      "auth",
		Description: "Scenario that requires a bearer credential, exercised with a malformed Authorization header.",
		Auth:        AuthSpec{Required: true, Token: SentinelToken},
		Headers:     sseHeaders,
		Wire:        successWire("never reached", "stop"),
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "A non-Bearer or empty credential is classified malformed and answers 401."},
	})
}

func registerRedirectScenarios() {
	Register(Scenario{
		Name:        "redirect-to-final-url",
		Family:      "redirect",
		Description: "302 redirect to a different path, which RadiChat must refuse rather than follow.",
		Redirect:    "/v1/chat/completions/stream-success-basic",
		Headers:     sseHeaders,
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "Redirects are refused; the client must not follow them."},
	})

	Register(Scenario{
		Name:        "redirect-with-query",
		Family:      "redirect",
		Description: "302 redirect whose Location carries a query string.",
		Redirect:    "/v1/chat/completions/stream-success-basic?token=placeholder",
		Headers:     sseHeaders,
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "A redirect with a query is still refused."},
	})

	Register(Scenario{
		Name:        "redirect-absolute",
		Family:      "redirect",
		Description: "302 redirect to an absolute loopback URL, which RadiChat must also refuse.",
		Redirect:    "http://127.0.0.1:1/v1/chat/completions/stream-success-basic",
		Headers:     sseHeaders,
		Mode:        ModeHTTP,
		Expect:      Expect{Method: "POST", Note: "Absolute redirect targets are refused too; the endpoint must be final."},
	})
}

func registerContextScenarios() {
	// These scenarios observe request shape. Each one asserts a documented
	// context-accounting or protocol property through Expect.

	Register(Scenario{
		Name:        "context-system-retained",
		Family:      "context",
		Description: "Fixture observing that the system message stays first across turns.",
		Headers:     sseHeaders,
		Wire:        successWire("ok", "stop"),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method:   "POST",
			Messages: []string{"system", "user"},
			Note:     "With a system prompt configured, the first request message is the system message.",
		},
	})

	Register(Scenario{
		Name:        "context-pair-trimming",
		Family:      "context",
		Description: "Fixture observing that trimming drops oldest complete user/assistant pairs.",
		Headers:     sseHeaders,
		Wire:        successWire("ok", "stop"),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method:   "POST",
			Messages: []string{"system", "user", "assistant", "user"},
			Note:     "After trimming, the remaining history is whole pairs: system, user, assistant, user.",
		},
	})

	Register(Scenario{
		Name:        "context-exact-budget-boundary",
		Family:      "context",
		Description: "Fixture whose request fits the budget exactly at the boundary.",
		Headers:     sseHeaders,
		Wire:        successWire("ok", "stop"),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Fields: map[string]any{"max_tokens": 1024},
			Note:   "max_tokens equals the configured generation_reserve, and the prompt fits exactly at the boundary.",
		},
	})

	Register(Scenario{
		Name:        "context-utf8-byte-accounting",
		Family:      "context",
		Description: "Fixture whose content is sized by UTF-8 bytes rather than rune count.",
		Headers:     sseHeaders,
		Wire:        successWire("ok", "stop"),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Fields: map[string]any{"stream": true},
			Note:   "Multibyte content is charged in bytes, so a rune-heavy prompt consumes more budget than its rune count suggests.",
		},
	})

	Register(Scenario{
		Name:        "context-failed-turn-rollback",
		Family:      "context",
		Description: "Fixture that fails mid-stream so the client must discard the staged turn.",
		Headers:     sseHeaders,
		Wire: []Segment{
			{Bytes: []byte(chunk("partial", ""))},
			{Bytes: []byte{}, CloseAfter: true},
		},
		Mode: ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Note:   "A failed stream must not commit the turn; the next request shows unchanged history.",
		},
	})

	Register(Scenario{
		Name:        "context-after-clear",
		Family:      "context",
		Description: "Fixture observing that /clear drops history but keeps the system prompt.",
		Headers:     sseHeaders,
		Wire:        successWire("ok", "stop"),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method:   "POST",
			Messages: []string{"system", "user"},
			Note:     "After /clear the next request carries only the system message and the new user message.",
		},
	})

	Register(Scenario{
		Name:        "context-max-tokens-sent",
		Family:      "context",
		Description: "Fixture asserting that max_tokens is sent and equals generation_reserve.",
		Headers:     sseHeaders,
		Wire:        successWire("ok", "stop"),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Fields: map[string]any{"max_tokens": 512, "stream": true},
			Note:   "max_tokens is present and equals the configured generation_reserve.",
		},
	})

	Register(Scenario{
		Name:        "context-no-optional-fields",
		Family:      "context",
		Description: "Fixture asserting that no tools, functions, or option bags are sent.",
		Headers:     sseHeaders,
		Wire:        successWire("ok", "stop"),
		Mode:        ModeHTTP,
		Expect: Expect{
			Method: "POST",
			Fields: map[string]any{"stream": true},
			Note:   "Only model, messages, stream, and max_tokens are sent; tools and functions are absent.",
		},
	})
}
