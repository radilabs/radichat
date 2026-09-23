# Phase 2 handoff

Date: 2026-09-23
Accepted: 2026-09-23
Status: **accepted**

## Stage / Phase

Stage 1 — Minimal Chat Client / Phase 2 — Endpoint Authentication.

## Accepted Baseline

The checkpoint commit containing this handoff is the accepted Phase 2 and Stage 1 baseline.

## Material Outcomes

RadiChat optionally reads `bearer_token_env` from its JSON configuration. The field names a process environment variable whose nonempty value is resolved at startup and sent as `Authorization: Bearer <token>` to the configured OpenAI-compatible endpoint. The config file and public example contain no credential value.

Omitting `bearer_token_env` preserves the accepted Phase 1 local-model path: no authentication header or credential is required, and its request, stream, transport, and diagnostic behavior remains covered by regression tests.

Authenticated error handling omits untrusted HTTP and stream bodies, redirect locations, and underlying transport details that could echo a credential. Authenticated streamed text filters complete token echoes across SSE deltas and trailing proper token prefixes. TLS remains strict and unchanged: system roots, TLS 1.2 minimum, no skip-verify, and no custom CA option. Redirects remain refused and requests are not retried.

## Findings Addressed

The first Dr Watson review identified credential-disclosure risks in authenticated response, redirect, transport, and streamed-output paths. These paths were hardened and received focused package and built-binary regression coverage. A second Dr Watson review passed. Independent Watcher verification passed all Phase 2 acceptance criteria and the Stage 1 goal.

## Tests / Evidence

- `gofmt -l .` and `git diff --check` — PASS.
- `go vet ./...` — PASS.
- `go test -timeout 180s -count=1 ./...` — PASS.
- `go test -timeout 180s -count=1 -race ./...` — PASS.
- Linux build and CLI help smoke — PASS.
- Authenticated and unauthenticated built-binary endpoint tests — PASS.
- Missing, empty, malformed, redirect, transport, HTTP-error, split-stream echo, and trailing-prefix credential tests — PASS.
- Independent Watcher seven-case built-binary endpoint harness — PASS.
- Filename-only credential-pattern scan of publishable files — no matches.

Runtime Coder, Dr Watson, and Watcher reports remain ignored under `reports/`.

## Owner Validation

The owner rebuilt and tested RadiChat against an authenticated endpoint, confirmed successful operation, accepted Phase 2, and authorized the checkpoint commit and push on 2026-09-23.

## Known Limitations

- The token is supplied through the process environment, which is not encrypted and may be observable to the same user or system administrator.
- Authenticated streaming conservatively redacts trailing assistant text that matches a proper prefix of the token; harmless matching text may therefore be replaced.
- Automated verification used controllable local endpoint fixtures; owner validation covered an authenticated endpoint.
- The supported OpenAI-compatible protocol subset remains deliberately narrow.

## Deferred Work

OAuth/login UI, credential storage and rotation, provider-specific authentication, TLS bypass/custom CA handling, persistence, databases, compaction, tools, agents, RAG, MCP, GUI/full-screen TUI, and non-Linux release targets remain outside the accepted scope.

## Notes for Next Phase

Stage 1 is complete. No later phase is authorized. Preserve the zero-auth path, environment indirection, strict TLS, generic authenticated diagnostics, and credential-echo regression coverage in any future work.
