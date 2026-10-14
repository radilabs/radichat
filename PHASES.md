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

# Future Phases

Planned concept only, not authorized or contracted:

* optional durable session storage;
* optional database-backed history;
* context compaction/summarization.

No future phase task files may be created until explicitly authorized.
