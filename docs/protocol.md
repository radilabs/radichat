# Protocol subset

RadiChat speaks a narrow slice of the OpenAI-compatible Chat Completions API.

## Request

`POST` to the configured URL with `Content-Type: application/json` and `Accept: text/event-stream`.

Body fields sent:

* `model` — configured model
* `messages` — staged `system` / `user` / `assistant` messages (`role` and `content` strings only)
* `stream` — always `true`
* `max_tokens` — configured `generation_reserve`

Not sent: `Authorization`, cookies, `tools`, `functions`, `tool_choice`, `n`, or other option bags.

The configured URL is used as-is. Environment HTTP(S) proxies are ignored so the request cannot silently change destination. Redirects are refused. Failed requests are not retried. Keep-alives and response compression are disabled.

TLS uses the system certificate pool and TLS 1.2 as a minimum. There is no insecure-skip-verify switch.

## Stream

Successful responses must be HTTP 2xx with `Content-Type` `text/event-stream` (or an omitted content type, which is still parsed as SSE). Other content types fail.

The decoder accepts:

* LF and CRLF line endings
* fragmented TCP reads, including multiple events in one read
* SSE comments (`:` lines) and ignored `event` / `id` / `retry` fields
* role-only chunks and empty deltas
* `data: [DONE]` as the terminal marker
* UTF-8 text in JSON string deltas, including multibyte characters

`delta.content` must be a JSON string when present. `finish_reason` values:

* success: omitted / empty, `stop`, or `length` (`length` means the server hit `max_tokens`; RadiChat still commits the complete streamed text and does not locally truncate)
* failure: `tool_calls`, `function_call`, `content_filter`, or any other nonempty reason

Tool-call / function-call deltas are errors. Stream JSON `error` objects fail as endpoint errors. A non-null `error` value that is not a JSON object (array, bool, string, number) fails as a malformed stream error payload. JSON `error: null` or an omitted `error` field is not an error. Premature EOF without `[DONE]`, malformed events, multiple choices, and oversized events fail.

Assistant deltas are written to stdout as they arrive. The consumer can observe text before the server ends the stream.

## Timeouts and limits

These are local RadiChat limits, not server tokenizer limits.

| Limit | Default |
| --- | --- |
| TCP dial | 10s |
| TLS handshake | 10s |
| Response headers | 15s |
| Idle TCP read during a stream | 60s (reset on each successful underlying body `Read` that returns bytes; a healthy slow or fragmented stream continues, a silent stall fails). Complete SSE events are not required to reset the watchdog. |
| No total request deadline | a long healthy stream is not cut off by a global timer |
| Config file | 1 MiB |
| SSE event | 1 MiB |
| Non-success HTTP body snippet | 8 KiB |
| Stream payload read | 8 MiB |
| Accumulated assistant UTF-8 bytes | `context_budget` (exceeding is an error; the turn is discarded, never silently truncated) |
| Input line | `max(context_budget, 4096)` bytes |

A stalled connection is cancelled and closed. An in-flight turn is not committed.

## Incompatible endpoints

Servers that require authentication, return non-streaming JSON, use a different event format, emit tool calls, or follow redirects will fail with a clear error. RadiChat does not probe alternate URLs or fall back to another protocol.
