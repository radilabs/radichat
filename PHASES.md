# RadiChat — Phase Contracts

These phases define immutable execution boundaries for RadiChat.

Tasks inside a phase may be refined as implementation progresses, but the phase goal, scope, exclusions, acceptance criteria, and handoff contract must not be changed during implementation.

Work discovered outside the active phase is recorded as deferred work and is not implemented.

Detailed task files are created only for the currently authorized phase.

---

# Phase Map

| Stage | Phase | Name | Outcome |
| ----- | ----- | ---- | ------- |
| 0 | 0 | Factory Bootstrap | Repository contains the Factory skeleton and agreed product contracts. |
| 1 | 1 | Local Chat Core | One Linux Go binary chats with an unauthenticated OpenAI-compatible endpoint using a config file. |
| 1 | 2 | Endpoint Authentication | The same client can optionally authenticate to compatible endpoints. |
| 2 | 3 | Factory v1.2 Migration | Accepted Factory v1.2 runtime and public/private boundary. |
| 3 | 4 | `radichat-mock` | Reusable standard-library mock server and reproducible test fixtures. |
| 3 | 5 | Adversarial protocol investigation | Bounded protocol investigation; accepted with zero confirmed divergences. |
| 4 | 6 | v0.1.1 File I/O and Batch Support | File-based system prompts, sequential file input, JSONL results, and Kubernetes Job documentation. |

Future capabilities require additional phase contracts.

---

# Stage 0 — Bootstrap

## Stage Goal

Create the governed repository from which implementation can begin without ambiguity.

## Stage Entry Conditions

* Repository exists.
* Product owner has defined the initial product boundary.

## Phases in This Stage

* Phase 0 — Factory Bootstrap

# Phase 0 — Factory Bootstrap (Stage 0)

## Goal

Populate RadiChat with the current Radilabs Factory skeleton and record the initial product and phase contracts.

## Entry Conditions

* Repository exists.
* Owner has explicitly requested Factory bootstrap.
* No implementation is required.

## Scope

* Factory role and lifecycle files.
* RadiChat product definition.
* Stage/phase contracts.
* Current execution state.
* Initial ignore rules.
* Phase 0 task file only.

## Explicit Exclusions

* Go source code.
* Product implementation.
* Phase 1 task file.
* Authentication.
* Session persistence.

## Acceptance Criteria

1. Required Factory skeleton files exist at repository root and standard directories exist.
2. `PROJECT.md` describes RadiChat as a minimal OpenAI-compatible terminal client and records initial non-goals.
3. `PHASES.md` defines Phase 1 as unauthenticated single-binary Linux chat and Phase 2 as authentication.
4. `TASKS.md` accurately reports Phase 0 as the only authorized detailed task.
5. No implementation or future phase task file is present.

## Handoff Contract

Before Phase 0 can be declared complete:

* all acceptance criteria are demonstrated;
* changed files and checks are recorded;
* known limitations and deferred work are recorded;
* Watcher verification has PASSed;
* owner explicitly accepts the phase.

Then STOP.

Do not begin Phase 1.

---

# Stage 1 — Minimal Chat Client

## Stage Goal

Deliver a genuinely small Linux terminal client that can talk to both local unauthenticated and authenticated OpenAI-compatible endpoints without adding persistence or agent behavior.

## Stage Entry Conditions

* Stage 0 accepted.
* Phase 1 explicitly authorized.

## Phases in This Stage

* Phase 1 — Local Chat Core
* Phase 2 — Endpoint Authentication

# Phase 1 — Local Chat Core (Stage 1)

## Goal

Deliver a single Linux Go binary that provides interactive streamed chat against an unauthenticated OpenAI-compatible endpoint using a config file.

## Entry Conditions

* Phase 0 accepted.
* Owner explicitly authorizes Phase 1.
* `tasks/phase-1.md` is created only after authorization.

## Scope

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

## Explicit Exclusions

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

## Acceptance Criteria

1. `go build` produces one runnable Linux executable.
2. With only the config file, the binary can connect to an unauthenticated OpenAI-compatible test/local endpoint and complete a streamed multi-turn chat.
3. Endpoint, model, context budget, generation reserve, and optional system prompt are configurable without recompilation.
4. Conversation state exists only in memory and can be cleared during the process.
5. The client stays within its configured working context budget using a documented deterministic trimming policy; it does not silently exceed the budget or invent persistence.
6. Connection/protocol/configuration errors fail clearly without hidden fallback.
7. Tests cover config parsing, request construction, streaming handling, and context trimming.
8. No Phase 1 exclusion is implemented.

## Handoff Contract

Before Phase 1 can be declared complete:

* all acceptance criteria are demonstrated with direct evidence;
* build/test commands and results are recorded;
* configuration and build/run instructions are documented;
* known limitations and deferred work are recorded;
* Watcher verification has PASSed;
* owner explicitly accepts the phase.

Then STOP.

Do not begin Phase 2.

# Phase 2 — Endpoint Authentication (Stage 1)

## Goal

Add optional authentication while preserving the zero-auth local-model path.

## Entry Conditions

* Phase 1 accepted.
* Owner explicitly authorizes Phase 2.
* `tasks/phase-2.md` is created only after authorization.

## Scope

* Optional bearer-token authentication for OpenAI-compatible endpoints.
* Secret supplied via environment/config indirection that avoids committing credentials.
* Authenticated and unauthenticated modes share the same chat behavior.
* Documentation and tests for both paths.

## Explicit Exclusions

* OAuth/login UI.
* Cloud-provider-specific credential workflows.
* Session persistence.
* Database.
* Compaction/summarization.
* Tools/agents/RAG/MCP.

## Acceptance Criteria

1. RadiChat can call a bearer-token-protected OpenAI-compatible endpoint.
2. Existing unauthenticated local endpoint behavior remains functional.
3. Credentials are not required for local use and are not printed in normal/error output.
4. Authentication behavior is covered by tests.
5. No Phase 2 exclusion is implemented.

## Handoff Contract

Before Phase 2 can be declared complete:

* all acceptance criteria are demonstrated;
* security-relevant behavior is documented;
* known limitations and deferred work are recorded;
* Watcher verification has PASSed;
* owner explicitly accepts the phase.

Then STOP.

---

# Stage 2 — Factory v1.2 Runtime

## Stage Goal

Run RadiChat development on Factory Core Concepts v1.2 with full persona coverage
and a clean public/private boundary.

## Stage Entry Conditions

* Stage 1 accepted (Phase 2 accepted).
* Owner explicitly authorizes Phase 3.

## Phases in This Stage

* Phase 3 — Factory v1.2 Migration

# Phase 3 — Factory v1.2 Migration (Stage 2)

## Goal

Migrate repository process structure to Factory Core Concepts v1.2 without
changing product behavior or accepted phase evidence.

## Entry Conditions

* Stage 1 accepted.
* Owner explicitly authorizes Phase 3.
* `tasks/phase-3.md` is created only after authorization.

## Scope

* Persistent-surface updates: `AGENTS.md`, `PROJECT.md` roles, `findings/`,
  `tasks/README.md` alignment, `.gitignore`.
* Untrack Factory runtime paths (`DRIVER.md`, `WATCHER.md`, `reports/README.md`).
* Materialize the ephemeral `.factory/` kit from a pinned Core Concepts commit and
  fill project-specific pipeline commands.
* Record the migration as a durable decision.

## Explicit Exclusions

* Go source code, product behavior, configuration formats.
* `PHASES.md` contracts of accepted phases (Phase 0/1/2 remain immutable).
* Historical `reports/` contents.
* New product features or future phase task files.

## Acceptance Criteria

1. `AGENTS.md` routes all v1.2 managed roles (Driver, Architect, Coder, Writer,
   Watcher) to `.factory/` entrypoints and retains the CAO transport section.
2. `PROJECT.md` roles include Architect and Writer/Documentation Steward.
3. `git ls-files` contains no `.factory/`, `reports/`, root `DRIVER.md`, or root
   `WATCHER.md` paths; `sh .factory/check-public-surface.sh` prints PASS.
4. `.factory/` contains role entrypoints, canonical contract snapshots,
   `defaults.yaml`, `agents.yaml`, filled `pipeline.yaml`, and `source.txt`.
5. The Core Concepts pin is an exact commit SHA (40 hex chars — never a branch,
   tag, or moving ref), recorded in `.factory/source.txt` **and** durably in
   `decisions/0002-factory-v1.2-migration.md`.
6. `decisions/` records the migration decision; `findings/README.md` exists.
7. `go build ./...`, `go vet ./...`, and `go test ./...` pass unchanged.
8. Accepted phase task files, handoffs, and decisions are unmodified.

## Handoff Contract

Before Phase 3 can be declared complete:

* all acceptance criteria are demonstrated with direct evidence;
* changed files and checks are recorded;
* known limitations and deferred work are recorded;
* Watcher verification has PASSed;
* owner explicitly accepts the phase.

Then STOP.

---

# Stage 3 — Protocol Verification Tooling

## Stage Goal

Provide reusable, reproducible test tooling for RadiChat's documented OpenAI-compatible streaming, authentication, error handling, and context-accounting behavior. Build the mock server in Phase 4; any adversarial investigation against RadiChat is a separate phase and requires separate Owner authorization.

## Stage Entry Conditions

* Stage 2 accepted.
* Owner explicitly authorizes a Stage 3 phase.

## Phases in This Stage

* Phase 4 — `radichat-mock` (authorized 2026-10-14)
* Phase 5 — Adversarial protocol investigation (authorized 2026-10-14)

# Phase 4 — `radichat-mock` (Stage 3)

## Goal

Build a reusable, standard-library Go mock server that can exercise RadiChat's documented OpenAI-compatible streaming, authentication, and error-handling behavior, with deterministic scenarios, focused tests, and usage documentation.

## Entry Conditions

* Stage 2 accepted.
* Owner explicitly authorizes Stage 3 / Phase 4.
* `tasks/phase-4.md` is created only after authorization.

## Scope

* A reusable Go mock-server package under `internal/mockllm/`, excluded from the RadiChat product dependency graph.
* A required `radichat-mock` development CLI under `cmd/radichat-mock/`, binding only to loopback by default, excluded from release and container artifacts.
* Deterministic, named OpenAI-compatible Chat Completions scenarios covering streamed success, framing/fragmentation, malformed or incomplete responses, HTTP errors, bearer authentication, redirects, and request/context-accounting fixtures.
* Safe request recording: method, path, relevant non-secret headers, decoded request fields, and authentication outcome; bearer-token values must never be retained or printed.
* Focused tests of the mock, scenario catalog, request recording/redaction, deterministic fragmentation, clean shutdown, and reproducible invocation; tests may use RadiChat only for a known-supported smoke path.
* Documentation for running the mock and its tests.

## Explicit Exclusions

* Any changes to RadiChat product code or accepted Phase 0–3 files/evidence.
* Running the full adversarial scenario matrix against RadiChat, defect triage, or durable defect findings. These require a separately authorized Phase 5.
* Product-code fixes, which require separate Owner authorization after finding selection.
* External services, real credentials, third-party Go dependencies, release/container inclusion, or showcase publication.
* Changes to the documented protocol, qualification rules, or existing product behavior.

## Acceptance Criteria

1. The mock uses only Go's standard library, binds only to loopback by default, and shuts down cleanly.
2. A deterministic named scenario catalog covers successful streamed responses; SSE framing and exact configured byte segmentation; malformed JSON/SSE and early termination; HTTP error statuses and non-SSE responses; bearer-token match/missing/mismatch; refused redirects; and request fixtures for system-message retention, complete-pair trimming, exact budget boundaries, UTF-8 byte accounting, failed-turn rollback, `/clear`, and `max_tokens`.
3. Fragmentation scenarios can split at fixed offsets inside UTF-8 sequences, CRLF, `data:` fields, and `[DONE]`. Tests prove the configured byte segments without claiming that TCP preserves write boundaries.
4. The recorder captures request method, path, relevant headers, decoded request fields, and auth outcome without retaining or printing bearer-token values. Tests use sentinel credentials only and assert they never appear in captured logs or command output.
5. The `radichat-mock` CLI supports listing and running named scenarios, emits its loopback URL, exits cleanly on interruption, and remains excluded from release/container artifacts. The root build still produces only the RadiChat binary.
6. Documentation provides reproducible commands for building/running the mock and running its tests. Full adversarial runs against RadiChat are explicitly deferred to Phase 5.
7. `go test ./... -count=1`, `go vet ./...`, `go build ./...`, `go run . -help`, and `sh .factory/check-public-surface.sh` pass. No third-party module is added; no non-test product-code file or accepted Phase 0–3 evidence changes.
8. The frozen Phase 4 candidate receives independent Watcher PASS against this contract, with actual repository state and direct evidence. Owner acceptance is still required to complete the phase.

## Handoff Contract

Before Phase 4 can be declared complete:

* all acceptance criteria are demonstrated with direct evidence;
* changed files, candidate identity, and check results are recorded in `tasks/phase-4.md`;
* known limitations and deferred Phase 5/6 boundaries are recorded;
* independent Watcher verification has PASSed;
* the evidence package is presented to the Owner, and explicit Owner acceptance is recorded.

Then STOP. Do not start Phase 5 or modify RadiChat product code.

# Phase 5 — Adversarial Protocol Investigation (Stage 3)

## Goal

Use the accepted `radichat-mock` tooling to conduct a bounded, reproducible adversarial investigation of RadiChat against its documented protocol behavior. Record verified divergences as durable findings without changing RadiChat product code or fixing defects.

## Entry Conditions

* Stage 2 and Stage 3 / Phase 4 accepted; Phase 4 checkpoint is `8e2951274e52f72963ad8ecb1d75a4a477fe6561`.
* Owner explicitly authorized Stage 3 / Phase 5 on 2026-10-14.
* `tasks/phase-5.md` is created after authorization and before test execution.

## Scope

* A test-only black-box harness under `internal/discovery/`, guarded by `//go:build mockdiscovery`, using `radichat-mock` and temporary configs/binaries.
* Harness-local `p5-` scenarios may be registered from test code when the accepted Phase 4 catalog lacks a needed payload. The accepted mock package, CLI, and product code are not changed.
* A bounded must-run matrix derived from `docs/protocol.md`, covering successful SSE parsing and fragmentation; malformed streams and early termination; HTTP errors and non-SSE responses; authentication and token redaction; redirects; and context-accounting boundaries and rollback.
* Reproducible per-case evidence under ignored `.factory/reports/` and one durable `findings/F-000N-*.md` per verified protocol divergence.
* Independent Luna 6 reproduction of confirmed findings plus additional cases selected independently from the documented contract; separate independent Watcher verification.

## Explicit Exclusions

* Any non-test RadiChat source change, product behavior change, defect fix, Phase 6 work, or changes to accepted Phase 0–4 evidence.
* Changes to `internal/mockllm/`, `cmd/radichat-mock/`, `docs/protocol.md`, or other product documentation to make an observed result appear compliant.
* Real credentials, external services, publishing, or showcase release. Use only the documented fake sentinel credentials.
* Treating undocumented behavior as a defect. Exploratory probes may be recorded in the task report but do not become findings unless separately authorized.
* A minimum defect count; zero verified divergences is a valid result.

## Acceptance Criteria

1. The tracked harness is test-only and build-tagged; the default `go test ./...` suite is unchanged. No product source, mock implementation, Phase 0–4 task/handoff/decision, or protocol contract is modified.
2. The must-run cases listed in `tasks/phase-5.md` execute with repeatable configs and exact case IDs. They cover stream success, fixed-fragment SSE parsing, malformed JSON/content and multiple choices, early EOF/truncation, supported stream-error/finish handling, HTTP 401/429/500/404, non-SSE responses, redirect refusal/no-follow, authenticated and unauthenticated behavior, token echo/redaction, request-field/header assertions, exact and over-budget UTF-8 byte boundaries, pair trimming, failed-turn rollback, `/clear`, and `max_tokens`.
3. `docs/protocol.md` and the linked context/CLI docs are the sole binding oracle. Each case records a verdict and the observed exit code, sanitized stdout/stderr evidence, and relevant request-record fields. Stderr wording not promised by the docs is advisory only. Sentinel values never appear in captured evidence or durable findings.
4. Every verified documented divergence has one durable `new` finding in `findings/`, with case/scenario ID, exact reproduction command, environment/config details excluding secret values, expected vs observed behavior, protocol citation, and supporting `.factory/reports/` evidence. No unverified hypothesis is reported as a defect; no findings are required.
5. Luna 6 uses a separate T3 delegated context to derive its own oracle before examining Worker results, reproduces each Worker finding from a clean state, uses an independent raw `net.Listener` harness for selected critical cases, and exercises at least three additional protocol-documented cases not selected by the Worker.
6. Dr Watson reviews the frozen Phase 5 candidate for scope, evidence quality, and omissions. The existing independent Watcher then verifies every acceptance criterion and reproduces the findings independently; it records explicit PASS or FAIL under `.factory/reports/`.
7. `go test -tags mockdiscovery ./internal/discovery -count=1 -run 'P5' -v`, default `go test ./... -count=1`, `go vet ./...`, `go build ./...`, `gofmt -l .`, `git diff --check`, and `sh .factory/check-public-surface.sh` produce recorded outcomes. Any known slow cases are opt-in and may be omitted to protect the deadline.
8. The final Phase 5 evidence package, findings (if any), Dr Watson review, independent Luna results, Watcher verdict, known limits, and an unpublished draft Discord showcase post are presented to the Owner by 2026-10-15 14:00 CEST. No publication occurs without separate Owner authorization.

## Handoff Contract

Before Phase 5 can be declared complete:

* the bounded matrix is run or any omitted case is explicitly identified with a reason;
* verified divergences and their reproducible evidence are recorded without fixes;
* accepted Phase 0–4 evidence and all product code are preserved;
* Luna 6 and Watcher independently report their results, and Dr Watson's review is resolved or listed as an open concern;
* the Owner receives the evidence package and draft showcase post.

Then STOP for Owner acceptance. Do not publish or begin any fix phase.

## Stage 4 — v0.1.1 File I/O and Batch Support

### Stage Goal

Deliver a small file-I/O and sequential batch capability for RadiChat while retaining the existing REPL and conversation model. Kubernetes support is documentation and examples only.

### Stage Entry Conditions

* Stage 3 / Phase 5 accepted.
* Owner approved design commit `334e9710efeab981782870df4bc0e050ef195be7` and explicitly authorized Stage 4 / Phase 6 on 2026-10-23.

### Phases in This Stage

* Phase 6 — v0.1.1 File I/O and Batch Support

### Stage Exit Conditions

* Phase 6 satisfies its acceptance criteria and receives independent Watcher PASS.
* Owner explicitly accepts Phase 6.
* Any later release, merge, or tag receives its own required Owner authorization.

# Phase 6 — v0.1.1 File I/O and Batch Support (Stage 4)

## Goal

Add file-based system prompts and opt-in sequential batch input/result output while preserving RadiChat's small, single-process REPL design.

## Entry Conditions

* Stage 3 / Phase 5 accepted.
* Owner approved the v0.1.1 design and authorized Stage 4 / Phase 6 on 2026-10-23.
* `tasks/phase-6.md` is created only after this authorization.

## Scope

* Add `system_prompt_file`, mutually exclusive with `system_prompt`, with config-relative path resolution, bounded UTF-8 file validation, and startup context-fit checks for both non-empty prompt sources. Configurations with no system prompt retain existing behavior.
* Add `-input PATH` to run the existing REPL sequentially from a file when stdin is unavailable. Preserve the existing stdin, line, `/clear`, `/quit`, EOF, failed-turn, and context-trimming semantics.
* Add opt-in JSONL results with `-output PATH` and JSONL-only stdout mode with `-output -`. Define and implement the approved v1 record schema, error/partial/interruption behavior, no-overwrite policy, output creation order, and secret redaction.
* Keep file opening/config-file I/O in startup/config loading. The REPL receives an `io.Reader` and a record-writer interface, coordinates conversation operations, and does not receive file paths. Dedicated input and JSONL components own bounded input reading and record encoding. `internal/chat` has no file I/O.
* Return finish reason through an additive client API that preserves existing callers where practical.
* Add focused unit/integration tests, update `PROJECT.md` Capabilities with one concise v0.1.1 bullet, update CLI/config/context-accounting documentation and README, add JSON-format Kubernetes Job/ConfigMap examples and documentation, and record durable decisions in `decisions/0003` including the startup-fit extension to decision 0001.
* Treat JSONL results as per-run output records only. They are independent of context trimming and are never trimmed when older turns leave the model context.

## Explicit Exclusions

* Database, conversation save/load or resume, implicit transcript storage, concurrency, retries, scheduling/orchestration, Kubernetes API/client code, Helm/operators, agents/tools, and new external dependencies.
* Changes to the supported inference protocol or unrelated RadiChat product behavior.
* Product fixes or deferred Phase 5 findings.
* Kubernetes runtime/deployment integration, real-cluster verification, container image build/publication, image release, merge, tag, or product release without separate Owner authorization.
* Any mutation or removal of accepted Phase 0–5 evidence/contracts. The historical Phase 5 exclusion referring to “Phase 6” described the then-planned product-fix phase; this Phase 6 is the separately approved Stage 4 feature phase.

## Acceptance Criteria

1. `system_prompt_file` is a non-empty JSON string, conflicts with `system_prompt` whenever both keys are present, resolves relative to the lexical config directory, follows symlinks, is a regular file no larger than 1 MiB, is valid UTF-8, strips one BOM and one terminal LF/CRLF, and rejects empty content. Errors do not reveal prompt bytes.
2. Both inline and file-sourced non-empty system prompts are checked at startup against the existing context fit policy with zero endpoint requests on failure. Configurations with no prompt keep existing behavior. The system prompt remains after `/clear`.
3. `-input PATH` is opened and validated before REPL execution, rejects missing paths/directories with exit 1 and no requests, uses existing line limits/semantics, accepts a regular file or FIFO, and runs without interactive prompts. With no new flags, stdin output is byte-identical except for the explicitly approved prompt-fit and SIGTERM changes.
4. JSONL v1 has exact field names/types and omission rules defined by the design and durable decision. One record is emitted for each handled nonblank physical input line; blank lines do not emit records, line numbers include blanks, and `/clear`, `/quit`, and command errors have defined records.
5. Successful records contain complete committed redacted output and finish reason. Failed or interrupted streams never emit success records or partial output; failed staged turns are aborted. `sent` is true only when the client observes a successfully written HTTP request (`httptrace.WroteRequest` with no error); otherwise it is false. Error records contain redacted diagnostics. An interruption record is best effort, with SIGINT exit 130 and SIGTERM exit 143 taking precedence.
6. JSONL output remains independent of context trimming: every processed result record remains in output even when its turn is later trimmed from conversation context. Tests verify this across multiple turns and `/clear` behavior.
7. `-output PATH` uses exclusive create and mode 0600, never appends/overwrites, and is created only after all other startup validation succeeds. Write failure exits 1 and prevents further requests. `-output -` emits only JSONL bytes to stdout and routes diagnostics to stderr. Existing use without new output flags retains terminal streaming behavior.
8. Authenticated-mode output and errors use existing redacted values; sentinel bearer values never appear in JSONL, stdout, or stderr. JSON records correctly round-trip quotes, slashes, control characters, HTML characters, and multibyte UTF-8.
9. Kubernetes Job/ConfigMap examples use JSON manifests validated with the standard library. Tests cross-check manifest args/mounts/data paths, parse embedded config with `config.Parse`, and assert a visibly marked placeholder image digest. Examples include non-interactive file input, stdout JSONL collection, read-only/non-root security, no automatic Job retry, and Secret-based token injection. Documentation makes clear that no real cluster or image execution was performed.
10. Existing stdin batch behavior and Phase 0–5 accepted evidence remain intact; no excluded feature or dependency is added. Required checks pass: `gofmt`, `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, `go build ./...`, CLI help smoke check, Kubernetes JSON structural checks, `git diff --check`, and `sh .factory/check-public-surface.sh`.
11. The exact candidate receives independent Watcher PASS against every criterion and the handoff contract. Owner acceptance remains required before the phase is complete or any checkpoint commit/tag/release occurs.

## Handoff Contract

Before Phase 6 can be declared complete:

* all acceptance criteria are demonstrated with direct evidence;
* changed files, candidate identity, commands and observed results are recorded in `tasks/phase-6.md`;
* known limitations and deferred work are recorded;
* independent Watcher verification has PASSed against the exact candidate;
* the evidence package is presented to the Owner and explicit Owner acceptance is recorded.

Then STOP. Do not merge, tag, or release without separate Owner approval.

# Future Phases

Planned concept only, not authorized or contracted:

* optional durable session storage;
* optional database-backed history;
* context compaction/summarization.

No future phase task files may be created until explicitly authorized.
