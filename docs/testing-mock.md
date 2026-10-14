# Testing RadiChat with `radichat-mock`

`radichat-mock` is a small development tool that serves deterministic, named
OpenAI-compatible Chat Completions scenarios on loopback. It exists so
RadiChat's documented streaming, authentication, error-handling, and
context-accounting behavior can be exercised repeatedly without a real model
server.

It is **not** part of the RadiChat product. It binds loopback only, writes
nothing to disk, and is not in the root package's build or dependency graph.

The mock lives in two places:

- `internal/mockllm/` — the reusable mock server and scenario catalog. Tests
  and other Go tooling can import it directly.
- `cmd/radichat-mock/` — the development CLI that lists and runs scenarios.

## Build

```bash
go build -o /tmp/radichat-mock ./cmd/radichat-mock
```

The mock uses only the Go standard library, like RadiChat itself.

## List scenarios

```bash
go run ./cmd/radichat-mock list
```

Scenarios are grouped by family:

| Family | What it covers |
| --- | --- |
| `stream` | Successful streamed responses, including multibyte UTF-8, CRLF framing, SSE comments, role-only chunks, `length` finish, omitted `Content-Type`, and empty completions. |
| `framing` | One stream split at a fixed byte offset inside a multibyte rune, a CRLF, a `data:` field, and `[DONE]`; byte-by-byte delivery; several events in one write. |
| `malformed` | Invalid JSON events, non-string `delta.content`, multiple choices, early termination before `[DONE]`, mid-event truncation, stream error objects and non-object error payloads, tool/function calls, and unsupported finish reasons. |
| `http-error` | HTTP 401, 429, 500, and 404 responses; a non-SSE `Content-Type` with a non-streaming body; an empty body. |
| `auth` | A scenario that requires a bearer credential, exercised with a matching token, no token, a wrong token, and a malformed header. |
| `redirect` | Refused redirects: relative, with a query string, and absolute. |
| `context` | Request fixtures for system-message retention, complete-pair trimming, exact budget boundaries, UTF-8 byte accounting, failed-turn rollback, `/clear`, `max_tokens`, and absence of optional fields. |

## Run a scenario

```bash
go run ./cmd/radichat-mock run stream-success-basic
```

The CLI prints the endpoint URL to put in `radichat.json`:

```text
scenario:   stream-success-basic
family:     stream
behavior:   Successful streamed response: one content delta, a stop finish reason, then [DONE].
auth:       none required
endpoint:   http://127.0.0.1:34589/v1/chat/completions/stream-success-basic
listening on loopback; press Ctrl-C to stop
```

It serves until you press Ctrl-C, then exits cleanly.

### Choose an address

```bash
go run ./cmd/radichat-mock run stream-success-basic -addr 127.0.0.1:8099
```

Only loopback addresses are accepted. `0.0.0.0:0` and other non-loopback hosts
are refused before binding:

```text
radichat-mock: refusing to bind a non-loopback address; use 127.0.0.1, [::1], or localhost
```

Flags may appear before or after the scenario name, so
`run stream-success-basic -addr 127.0.0.1:8099` works too.

## Chat with RadiChat against a scenario

Use two terminals. In terminal A, start the mock on the fixed loopback address
used by the example config below. It stays in the foreground:

```bash
go run ./cmd/radichat-mock run stream-success-basic -addr 127.0.0.1:34589
```

In terminal B, use the example config below. Its `endpoint` matches the
address above. If you chose a different `-addr`, change the port in the config
to match, or copy the endpoint URL the CLI prints.

To stop the mock, press Ctrl-C in terminal A.

Save this as `radichat.local.json` in the repository root (the file is ignored
by Git):

```json
{
  "endpoint": "http://127.0.0.1:34589/v1/chat/completions/stream-success-basic",
  "model": "mock-model",
  "context_budget": 8192,
  "generation_reserve": 1024,
  "system_prompt": "You are a concise assistant."
}
```

In terminal B, from the repository root:

```bash
go run . -config radichat.local.json
```

Type a message and press Enter. Type `/quit` to exit RadiChat. Then press Ctrl-C
in terminal A to stop the mock.

### Authenticated scenarios

`auth-*` scenarios require a bearer token. Export the sentinel in terminal B
and name the variable in the config. The sentinel is an obviously fake value;
never use a real credential with this mock.

Stop the previous mock (Ctrl-C in terminal A), then start the authenticated
scenario in terminal A on the same fixed address:

```bash
go run ./cmd/radichat-mock run auth-required-match -addr 127.0.0.1:34589
```

In terminal B:

```bash
export MOCKLLM_SENTINEL_TOKEN='mockllm-sentinel-token-not-a-real-credential'
```

Save this as `radichat.local.json`:

```json
{
  "endpoint": "http://127.0.0.1:34589/v1/chat/completions/auth-required-match",
  "model": "mock-model",
  "context_budget": 8192,
  "generation_reserve": 1024,
  "bearer_token_env": "MOCKLLM_SENTINEL_TOKEN"
}
```

Run `go run . -config radichat.local.json` as above. To drive the
`auth-required-missing` and `auth-required-mismatch` cases from RadiChat, change
what RadiChat sends: remove `bearer_token_env` from the config for the missing
case, or export a different value, such as
`mockllm-mismatch-token-not-a-real-credential`, for the mismatch case. The CLI
accepts the sentinel for every `auth-required-*` scenario; the scenario itself
decides whether the request matched.

`auth-malformed-header` requires a raw HTTP client such as curl; RadiChat cannot
drive it. RadiChat validates `bearer_token_env` before it sends anything and
refuses any value that is not a single-line HTTP token, so it never sends a
malformed `Authorization` header.

Stop the running mock (Ctrl-C in terminal A), then start the malformed-header
scenario in terminal A:

```bash
go run ./cmd/radichat-mock run auth-malformed-header -addr 127.0.0.1:34589
```

In terminal B, send a request with an empty bearer token:

```bash
curl -i -X POST http://127.0.0.1:34589/v1/chat/completions/auth-malformed-header \
  -H 'Content-Type: application/json' -H 'Authorization: Bearer ' -d '{}'
```

The request is answered with HTTP 401. The recorder classifies it as `malformed`
(see the next section); that record is in memory and is visible only from Go
tests.

## Request recording and credential redaction

Every request is recorded in memory. A record contains:

- HTTP method and URL path (without the query string)
- selected request headers (`Content-Type`, `Accept`), with `Authorization`
  replaced by the placeholder `<redacted-authorization>`
- the decoded JSON request body (`Record.Body`), for example `model`,
  `messages`, `stream`, and `max_tokens`
- the full raw request body text (`Record.BodyRaw`), as sent
- the classified authentication outcome (`Record.Auth`), reported by
  `AuthOutcome.String()` as one of `none`, `malformed`, `match`, or `mismatch`.
  The underlying boolean fields are `Present` (an `Authorization` header was
  sent), `Malformed` (the header was not `Bearer <token>` with a nonempty
  token), and `Matched` (the token equals the server's configured expected bearer token)
- the scenario that handled the request

The mock never stores the value of the `Authorization` header. The recorder
keeps only the classification, so the bearer token itself does not remain in
memory after the request is handled.

That guarantee covers the header only. `Record.BodyRaw` and `Record.Body` hold
the request content exactly as the client sent it, in memory, for the life of
the process. If a caller puts a credential in a message or any other body field,
the recorder keeps that credential. Do not put real credentials, or any value
you need to keep secret, in prompts or request bodies. Use only the obviously
fake sentinel values documented here.

Cookies are omitted entirely. The recorder does not keep the `Cookie` header, a
truncated value, or a redacted marker, because a cookie can carry a credential.
To assert that no cookie was sent, check that the `Cookie` key is absent from
`Record.Headers`.

The query string is not captured at all. `Record.Path` holds only the URL path,
because a query string can carry a credential such as `?access_token=...`.

In Go tests:

```go
srv, err := mockllm.Start(mockllm.Options{ExpectedBearer: mockllm.SentinelToken})
if err != nil {
    return err
}
defer srv.Close()

// ... drive a request against srv.URL() + scenario.Path() ...

rec, ok := srv.Last()
if !ok {
    return errors.New("no request recorded")
}
if !rec.Auth.Matched {
    return errors.New("expected a matched credential")
}
```

## Deterministic fragmentation

Fragmentation scenarios write one configured byte segment per write. The
segments are the deterministic unit: the same scenario always produces the same
segments in the same order.

This is a statement about what the mock writes, not about what a reader
observes. TCP and the operating system may coalesce or split writes, so a test
must assert the configured segments and, separately, that a client reassembles
the concatenated bytes correctly. It must not assert that a reader sees one
segment per read.

The `fragment-*` scenarios split inside a multibyte UTF-8 rune, between the CR
and LF of a line ending, inside a `data:` field, and inside the `[DONE]` marker.
`fragment-byte-by-byte` delivers one byte per write, and
`fragment-multiple-events-one-write` delivers several complete events in a
single write, which is the opposite case.

## Running the tests

```bash
go test ./internal/mockllm/ -count=1 -v
go test ./cmd/radichat-mock/ -count=1 -v
```

Or run everything:

```bash
go test ./... -count=1
```

The mock's own tests cover loopback-only binding, clean shutdown, scenario
reachability, deterministic segmentation, request recording and credential
redaction, and CLI behavior. A small number of them also drive the real
RadiChat binary against a known-supported scenario to confirm the mock is
usable end to end.

## Scope boundary

Running the full adversarial scenario matrix against RadiChat, triaging any
defects it finds, and recording durable findings are **Phase 5** work. They
require separate Owner authorization after Phase 4 is accepted.

Merely defining or running a scenario here does not authorize defect triage. If
a scenario reveals unexpected RadiChat behavior, capture it for the Driver
rather than fixing product code, and do not create a `findings/` record under
this phase.

## Limitations

- The mock implements the documented protocol subset in
  [Protocol subset](protocol.md). It is not a general-purpose model server.
- `context-*` fixtures assert the documented byte-based accounting policy in
  [Context accounting](context-accounting.md). They do not measure real model
  tokens.
- The mock has no TLS. It serves plain HTTP on loopback, so it cannot exercise
  RadiChat's HTTPS behavior.
- The mock is a development tool. The root package's build and dependency graph
  exclude it, and `go build .` still produces only RadiChat. This repository
  contains no release or container build definitions to inspect, so the mock's
  absence from release binaries and the container image is not verified here.
  The mock was not added to any release or container integration. Check that
  separately wherever those artifacts are built.
