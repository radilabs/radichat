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

# Future Phases

Planned concept only, not authorized or contracted:

* optional durable session storage;
* optional database-backed history;
* context compaction/summarization.

No future phase task files may be created until explicitly authorized.
