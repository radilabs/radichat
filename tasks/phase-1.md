# RadiChat — Stage 1 / Phase 1: Local Chat Core

## Authorization and Status

Status: **accepted**.

Owner explicitly approved Phase 1 execution and the local context-accounting choice, then accepted the verified result on 2026-09-21. Implementation, review, and verification used separate delegated roles. Phase 2 remained unauthorized at this checkpoint.

## Accepted Baseline and Read First

Accepted Phase 0 checkpoint: `963a6d252f536a9c38babbd0159b1f341972cd30`. Record any subsequent planning checkpoint separately at execution startup.

Read `AGENTS.md`, `PROJECT.md`, `PHASES.md`, `TASKS.md`, `tasks/README.md`, `docs/handoffs/phase-0.md`, and this file. Driver and Watcher also read their respective role contracts.

## Immutable Contract Reference

Authority: `PHASES.md` → Stage 1 / Phase 1. The following is copied for execution convenience; design proposals below do not amend it.

### Goal

Deliver a single Linux Go binary that provides interactive streamed chat against an unauthenticated OpenAI-compatible endpoint using a config file.

### Scope

* Go implementation.
* Single Linux binary.
* Config file containing at minimum endpoint and model.
* Configurable context budget and generation reserve.
* Optional system prompt.
* OpenAI-compatible `/v1/chat/completions`.
* Streaming assistant output.
* Multi-turn conversation kept only in process memory.
* Clean exit and clear error reporting.
* Minimal local commands required for usable chat, such as clearing in-memory context and quitting.
* Build and focused tests sufficient to verify protocol/config/context behavior.

### Explicit Exclusions

* Authentication headers, API keys, OAuth, or credential storage.
* Session saving/loading.
* Database.
* Compaction/summarization.
* Tools/function calling.
* Agents.
* RAG/embeddings.
* MCP.
* GUI/full-screen TUI.
* Non-Linux release targets.

### Acceptance Criteria

1. `go build` produces one runnable Linux executable.
2. With only the config file, the binary can connect to an unauthenticated OpenAI-compatible test/local endpoint and complete a streamed multi-turn chat.
3. Endpoint, model, context budget, generation reserve, and optional system prompt are configurable without recompilation.
4. Conversation state exists only in memory and can be cleared during the process.
5. The client stays within its configured working context budget using a documented deterministic trimming policy; it does not silently exceed the budget or invent persistence.
6. Connection/protocol/configuration errors fail clearly without hidden fallback.
7. Tests cover config parsing, request construction, streaming handling, and context trimming.
8. No Phase 1 exclusion is implemented.

### Handoff Contract

* all acceptance criteria are demonstrated with direct evidence;
* build/test commands and results are recorded;
* configuration and build/run instructions are documented;
* known limitations and deferred work are recorded;
* Watcher verification has PASSed;
* owner explicitly accepts the phase.

Then STOP. Do not begin Phase 2. Phase 1 completion does not complete Stage 1, which also contains Phase 2.

## Approved Design

### Structure and Dependencies

Use Go, Linux, and the standard library unless direct evidence justifies a small dependency within scope. Keep a root executable entry point so plain `go build` works. Separate config validation, conversation budgeting, HTTP/stream decoding, and the REPL behind small functions/packages; avoid plugin interfaces and generic provider frameworks. Select and document the available supported Go toolchain at preflight rather than inventing a version during planning.

Suggested layout: `main.go`, `internal/config/`, `internal/chat/`, `internal/client/`, and `internal/repl/`, with focused tests alongside implementation. Packages may be combined where separation adds no value.

### Configuration and Invocation

Propose a strict JSON config at `./radichat.json`, with an optional `-config PATH` selector and `-help`. A default launch reads that file and needs no endpoint/model CLI flags, environment settings, login, or discovery call. Ship a documented example, not a live user config.

Fields: `endpoint` (full HTTP(S) chat-completions URL, e.g. `http://127.0.0.1:8080/v1/chat/completions`), `model` (required nonempty string), `context_budget` (positive integer), `generation_reserve` (positive integer smaller than the budget), and optional `system_prompt` (string). Example budgets: 8192 total and 1024 generation reserve. Require both budget fields so their values are explicit.

Reject unreadable/missing files, malformed JSON, duplicate or unknown keys, trailing JSON values, wrong types, invalid budgets, and unusable URLs. Reject endpoint user-info credentials, fragments, and query strings; introduce no authentication fields or headers. Send the configured URL directly without guessing or appending paths. Disable automatic redirects and retries so requests do not silently switch endpoints or duplicate turns. Error messages should identify the problem and corrective action without dumping whole config files or request bodies.

### Conversation and Context Accounting

Conversation consists of an optional fixed system message followed by completed user/assistant pairs. Before a request, stage the new user message, reserve generation capacity, and remove the oldest complete pairs until the proposed request fits. Preserve the system message and current user message. If those alone do not fit, fail locally without contacting the endpoint or changing committed conversation state. Print a brief notice when older turns are removed.

Proposed deterministic accounting: charge UTF-8 content bytes plus a fixed per-message and per-request allowance, documented as application budget units. Initial proposal: 32 units per message and 32 per request. Require `estimated_prompt_units + generation_reserve <= context_budget` for every request, including model-returned assistant content on later turns. Do not serialize JSON escape overhead as conversation content. Use overflow-safe arithmetic.

**Owner-approved accounting interpretation:** this estimator enforces the application's working budget but cannot guarantee exact server tokenizer/context-template usage for arbitrary models. Do not label it an exact token count or promise a universal model-context bound. If AC5 is intended to require exact server-token enforcement, stop for an owner-approved tokenizer/model strategy before coding this part; do not silently substitute an approximation. A server context rejection remains an explicit error, with no hidden retry or fallback. Owner approval explicitly resolved this interpretation in favor of deterministic local estimation.

Only commit the trimmed history, new user message, and complete assistant response after a successful stream. Partial output may be displayed, but errors/cancellation discard the staged turn and preserve the previous committed conversation. `/clear` removes conversation history and retains the configured system prompt. Bound any single stream's accumulated output with a documented local limit derived from the configured budget; never silently truncate a successful response or retain an unbounded buffer.

### HTTP and Streaming

POST a JSON Chat Completions request containing the configured model, staged messages, `stream: true`, and the generation cap using the field supported by the declared compatibility target (`max_tokens` proposed). Send no Authorization header, cookies, tools, or function definitions. Document the supported protocol subset after verifying it during execution; incompatible endpoints fail clearly.

Use an incremental SSE decoder that handles network fragmentation, CRLF/LF, comments/blank lines, event boundaries, role-only chunks, text deltas, and the terminal `[DONE]` marker. Preserve UTF-8 output without waiting for the whole response. Treat malformed events, error objects, unexpected payloads, and premature EOF as errors. Define successful completion and non-success finish reasons explicitly; tool-call responses are unsupported and must not be silently accepted. Bound response error bodies and event sizes; close bodies and cancel requests on all exit paths. Establish finite connection/header and idle-read timeouts without a short total deadline that cuts off a healthy long stream.

### Terminal Behavior

Provide a plain line-oriented REPL with `/clear` and `/quit`; blank input sends no request. Unknown slash commands report an error rather than being sent accidentally. EOF exits cleanly. Ctrl-C cancels an in-flight request and exits cleanly, discarding its staged turn. Keep assistant text on stdout and diagnostics/notices on stderr; avoid decorative/full-screen UI and ANSI control sequences in redirected output. Show interactive prompts only when appropriate. Bound input explicitly and report overlong lines clearly.

Invalid startup config exits nonzero. Recoverable per-turn network/protocol errors leave the interactive REPL usable; document exit behavior for redirected input and interrupted runs. No history files, transcript logs, autosave, or background service.

## Authorized Execution Tasks

### Task 1.0 — Driver preflight and design resolution

- [x] Record explicit owner execution authorization and full accepted baseline; inspect pre-existing changes.
- [x] Resolve AC5 estimator versus exact token-accounting interpretation with the owner before implementation. Record the accepted policy as a durable decision.
- [x] Inspect Linux/Go tooling, build/cache paths, local config and secret exposure, and required endpoint/test transport without printing credentials.
- [x] Update `.gitignore` before workers run for the default live config, generated binaries, coverage/build outputs, and applicable local runtime files; keep a safe sample config tracked.
- [x] Verify CAO availability/provider support; delegate through CAO when supported, otherwise record the permitted fallback. Do not assume prior CAO unavailability persists.
- [x] Assign bounded coder responsibilities and explicit evidence requirements; confirm worker CLI/model selection at launch. No worker is launched by draft approval alone unless execution is explicitly authorized.

### Task 1.1 — Executable skeleton and validated config (AC1, AC3, AC6, AC7, AC8)

Depends on 1.0.

- [x] Create the minimal Go module and root entry point, with documented build/run/help behavior.
- [x] Implement config loading/validation and safe example configuration.
- [x] Test valid values, missing fields/files, unknown/duplicate keys, trailing input, invalid URL/budget/types, and clear startup errors.
- [x] Demonstrate the built binary starts using only a config file; no credentials or daemon required.

### Task 1.2 — Deterministic in-memory conversation (AC3–AC7)

Depends on 1.1 and the approved AC5 policy.

- [x] Implement staged turns, deterministic accounting, generation reservation, oldest-complete-pair trimming, and clear/reset.
- [x] Preserve system/current user; reject an oversized mandatory prompt without mutation or HTTP traffic.
- [x] Commit only successful complete responses; rollback cancellation/protocol failures including after partial output.
- [x] Test exact accounting boundaries, repeated trimming, Unicode/multibyte text, large reserves, overflow, clear behavior, failed-turn rollback, and deterministic repeated input.
- [x] Add invariant/property coverage where useful: every admitted request fits the chosen accounting bound, retained turns are ordered complete pairs, and protected messages are retained.

### Task 1.3 — Unauthenticated client and streaming decoder (AC2, AC5–AC8)

Depends on 1.1; integrate with 1.2 before completing this task.

- [x] Implement the declared HTTP request subset and streaming parser with cancellation, bounded reads, timeout behavior, and no redirects/retries.
- [x] Use a local Go HTTP test server to inspect method/path/headers/body/model/messages/reserve and absence of authentication or excluded features.
- [x] Test fragmented SSE, Unicode boundaries, multiple events per read, comments, empty deltas, termination, malformed JSON/events, premature EOF, non-2xx errors, unsupported tool output, and interrupted streams.
- [x] Prove a stalled response terminates and releases resources; distinguish idle stalls from ongoing streaming.
- [x] Prove text reaches the consumer before the server ends the stream, using explicit synchronization rather than a timing-only assertion.

### Task 1.4 — REPL integration and black-box behavior (AC1–AC8)

Depends on 1.2 and 1.3.

- [x] Connect input, budgeting, streaming output, clear, quit, EOF, and cancellation.
- [x] Verify oversized/blank/command input sends no unintended request; document slash-command behavior.
- [x] Exercise the built binary against a controllable local endpoint using only config plus stdin: two streamed turns, observed history, trimming, clear, failed turn, recovery, and quit.
- [x] Demonstrate Ctrl-C termination and no hangs at input, connection, or streaming boundaries.
- [x] Verify a second process starts with no previous conversation and no transcript/storage artifacts are created.

### Task 1.5 — Documentation, evidence, and review (AC1–AC8; handoff)

Depends on 1.4.

- [x] Document Linux build/run, complete config schema/example, endpoint URL semantics, protocol subset, commands, errors, timeouts/limits, and approved context accounting with limitations.
- [x] Record `go build`, `go test ./...`, and appropriate static checks such as `go vet ./...`, with actual results and toolchain/environment. Run targeted additional tests only for observed risks.
- [x] Record black-box commands/results and map each AC to direct evidence, not worker completion claims.
- [x] Request Dr Watson review for networking, stream/state handling, cancellation, and budgeting risks. Route in-scope findings to coders; defer scope expansion.
- [x] Record changed files, limitations, deferred work, decisions, review disposition, and reproducible checks.

### Task 1.6 — Independent verification and owner gate

Depends on passing local checks and completed implementation evidence.

- [x] Launch independent Watcher with active contract, diff, task evidence, and relevant reviewer findings.
- [x] Obtain explicit PASS/FAIL for every criterion, handoff readiness, exclusions, and actual repository state.
- [x] Route any in-scope FAIL back for correction, testing, and independent re-verification. Driver never substitutes for Watcher.
- [x] Immediately before presenting for acceptance, ensure Watcher result applies to the final implementation; re-verify after corrections.
- [x] Present evidence, limitations, deferred work, reviewer concerns, and Watcher result to owner; STOP for explicit Phase 1 acceptance.
- [x] Only after acceptance, record accepted status, write `docs/handoffs/phase-1.md`, promote durable knowledge, and create the appropriate checkpoint.
- [x] STOP. Do not create Phase 2 tasks or treat Stage 1 as complete.

## Verification Matrix

| Criterion | Required direct evidence |
| --- | --- |
| AC1 | Plain `go build`; execute the resulting Linux binary. |
| AC2 | Built-binary multi-turn streamed exchange with local unauthenticated test endpoint and config only. |
| AC3 | Config tests plus observed request/behavior changes for every configurable field without recompilation. |
| AC4 | Captured requests before/after clear; new-process reset; no persistence artifacts/code. |
| AC5 | Approved accounting policy, boundary/invariant tests, captured requests demonstrating trimming and reserve, oversized-prompt rejection before network. |
| AC6 | Config/network/protocol/stall/cancellation failure tests; clear diagnostics; no hidden fallback or corrupted history. |
| AC7 | Passing focused config/request/stream/context tests with recorded commands. |
| AC8 | Independent diff/tree/dependency/request review against every exclusion. |

## Progress and Evidence

Implementation and reviewer correction loop complete. The frozen corrected source passed final independent Watcher verification. Full tests, focused regressions, vet, Linux build, library race checks, eight general black-box probes, and one fragmented-stream binary probe passed. See the Verification Record below for durable evidence; detailed runtime reports remain ignored.

## Changed Files

Lifecycle: `tasks/phase-1.md`, `TASKS.md`, `tasks/README.md`; ignore rules and `README.md`; `go.mod`, `main.go`, `main_test.go`; config/chat/client/repl source and tests under `internal/`; `radichat.example.json`; four technical documents under `docs/`; approved accounting decision under `decisions/`. Runtime reports are ignored. No accepted contract or Phase 0 implementation artifact changed.

## Known Limitations / Open Decisions

- Local byte-based accounting is approved; it does not guarantee exact server tokenizer/template usage.
- JSON format, full endpoint URL, default config path, and command/error behavior are approved for execution.
- Supported protocol subset, timeout/size limits, and errors are recorded in `docs/protocol.md`; authentication and broader endpoint compatibility remain excluded.
- Runtime verification uses local test endpoints; no live model endpoint smoke test is claimed.

## Deferred Work

Phase 2 authentication; later persistence, database/history, compaction, tools/agents, RAG, MCP, GUI/TUI, and non-Linux targets. No tasks for these are authorized here.

## Decisions / Regression Memory

Approved local accounting is recorded in `decisions/0001-phase-1-context-accounting.md`. Record confirmed-defect regression protection as discovered.

## Watcher / Review Status

Dr Watson completed attempt 01; three findings were dispositioned in the coder correction report. A stale-context verification attempt was superseded. The final Watcher independently verified the frozen corrected hashes and returned explicit PASS for all acceptance criteria and handoff items within its authority.

## Handoff Status

**ACCEPTED — OWNER ACCEPTED PHASE 1 ON 2026-09-21.**

The accepted checkpoint includes `docs/handoffs/phase-1.md`. Phase 2 remains unauthorized.

## Verification Record

- The Coder Team implemented the accepted design and supplied source, test, build, and documentation evidence. The Driver independently inspected the repository and reproduced the required checks.
- Initial review found an idle-stream watchdog defect and stream-error/configuration clarity issues. The implementation was corrected with focused regressions; an incorrect proposal to permit JSON `null` for optional strings was rejected against the approved contract.
- The corrected tree passed formatting, vet, Linux build, the full test suite, race checks, eight built-binary endpoint probes, and a fragmented-stream probe.
- The final independent Watcher verified the frozen corrected source and returned PASS for AC1–AC8, exclusions, and every handoff condition within its authority.
- Owner accepted Phase 1 on 2026-09-21. The accepted public checkpoint contains no credentials or private endpoint configuration; tracked examples use loopback placeholders and runtime reports remain ignored.
