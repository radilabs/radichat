# Phase 1 handoff

Date: 2026-09-21
Accepted: 2026-09-21
Status: **accepted**

## Stage / Phase

Stage 1 — Minimal Chat Client / Phase 1 — Local Chat Core.

## Accepted Outcome

RadiChat is a small standard-library Go terminal client for unauthenticated OpenAI-compatible chat-completions endpoints. It builds as one Linux executable, reads a strict JSON configuration file, streams assistant output, retains multi-turn history only in memory, supports `/clear` and `/quit`, and fails clearly on configuration, transport, and protocol errors.

The checkpoint commit containing this handoff is the accepted Phase 1 baseline.

## Configuration and Protocol

The configured endpoint is the complete chat-completions URL and is used without path discovery, redirects, retries, authentication headers, or credentials. Configuration includes model, context budget, generation reserve, and an optional string system prompt. `radichat.example.json` contains only safe loopback placeholders.

The supported streaming subset and limits are documented in `docs/protocol.md`; build, run, command, and exit behavior are documented in `README.md` and `docs/cli.md`.

## Context Accounting

The accepted policy uses deterministic application budget units: UTF-8 content bytes plus 32 units per message and 32 units per request, with the configured generation reserve held back. The client removes oldest complete user/assistant pairs while preserving the system prompt and current user message. It rejects an oversized protected prompt before network access. This is a local estimate, not an exact model-tokenizer guarantee.

## Verification

Independent final Watcher verification passed all eight Phase 1 acceptance criteria on the frozen corrected source. Formatting, static analysis, Linux build, the full test suite, library race checks, eight end-to-end black-box probes, and a separate fragmented-stream probe passed. Reviewer findings concerning idle-read activity and stream-error classification were corrected with regression coverage; strict optional-string handling was clarified and tested.

The public-release checkpoint was separately scanned for secrets, credentials, private endpoints, local paths, generated artifacts, and excluded Phase 2 behavior before commit. Runtime reports remain ignored.

## Known Limitations

- Context accounting does not reproduce model-specific tokenization or server chat-template overhead.
- Verification used controllable local test endpoints; no live model-service smoke test is claimed.
- The supported OpenAI-compatible protocol surface is deliberately narrow and documented.

## Deferred Work

Phase 2 authentication is not authorized. Persistence, databases, compaction, tools, agents, RAG, MCP, GUI/full-screen TUI, and non-Linux release targets remain deferred.

## Next Action

Stop at this checkpoint. Do not create or begin Phase 2 without explicit owner authorization.
