# RadiChat

RadiChat is a lightweight terminal client for chatting with local or remote OpenAI-compatible language-model endpoints.

The product goal is intentionally narrow: start fast, consume little context, have no agent machinery, and remain useful with small local context windows such as 8k.

---

## Product Model

RadiChat manages:

* one configured OpenAI-compatible endpoint;
* one active model;
* one in-memory conversation for the lifetime of the process;
* terminal input/output and streamed assistant responses.

The application owns its terminal interaction and in-memory chat state.

The model server owns inference. RadiChat does not attempt to become an agent harness.

## Capabilities

### Terminal chat

* interactive REPL;
* streamed assistant output;
* multi-turn conversation held in memory;
* basic local commands for process/session control where required by the active phase.

### Configuration

* configuration file for endpoint, model, context budget, generation reserve, and system prompt;
* Phase 1 assumes no authentication;
* Phase 2 adds optional endpoint authentication.

## Design Direction

Boring on purpose.

RadiChat should behave like a good Unix tool: quick to install, quick to understand, and difficult to misconfigure. Terminal output should remain readable without requiring a full-screen TUI.

No decorative interface or framework is justified unless a later phase contract explicitly requires it.

## Surfaces

* **Terminal REPL** — primary user interface.
* **Configuration file** — persistent user configuration.
* **CLI flags** — only where they materially improve bootstrap or override behavior.

## Technical Direction

* Go;
* Linux first;
* single distributable binary;
* OpenAI-compatible Chat Completions API;
* streaming responses;
* minimal dependencies;
* no required daemon or companion service.

## Product Principles

* Small context windows are first-class.
* Storage and model context are separate concerns.
* Features do not enter the binary merely because an SDK makes them easy.
* Local-model use must not require authentication.
* Clear failure is preferable to hidden fallback.
* The default path should need no database and no agent framework.

## Initial Non-Goals

The initial product does not include:

* session saving or loading;
* databases;
* conversation compaction or summarization;
* tools or function calling;
* agents or multi-agent orchestration;
* embeddings or RAG;
* MCP;
* browser or shell execution;
* GUI/TUI framework;
* non-Linux release targets;
* authentication in Phase 1.

These may be reconsidered only through later phase contracts.

---

# Development Factory Rules

The execution hierarchy is:

**Project → Stage → Phase → Task**

## Stage Boundaries

A stage groups one or more phases into a coherent product milestone or development state.

Each stage in `PHASES.md` defines its goal and exit conditions. Stage exit conditions must be checked explicitly and accepted by the project owners.

## Phase Boundaries

A phase is an immutable execution boundary once authorized.

Tasks may be refined inside the active phase, but its goal, scope, exclusions, acceptance criteria, and handoff contract must not be expanded during implementation.

Work discovered outside the active phase is recorded as deferred work. It is not implemented.

Watcher PASS does not by itself complete a phase. Project-owner acceptance is required.

No role may begin the next phase automatically.

## Task Availability

`PHASES.md` contains the stage roadmap and immutable phase contracts.

Detailed task files are created only for the currently authorized phase.

Future phase task files must not be created in advance.

## Evidence

Model narration, intent, or a worker saying "done" is not evidence.

Progress and verification use direct observations such as repository state, diffs, builds, tests, runtime behavior, logs, and independent Watcher observations.

**Model proposes actions; the harness owns truth.**

## Roles

### Owner / Roboticist

Defines intent, evaluates the actual product, and authorizes/accepts stage and phase transitions.

### Planner

Maintains project structure, stage/phase contracts, architecture boundaries, and acceptance criteria.

### Driver / Orchestrator

Coordinates execution according to `DRIVER.md`.

### Reviewer / Dr Watson

Inspects implementation, questions assumptions, identifies risks, and proposes deferred work.

### Coder Team

Implements only the authorized phase.

### Verifier / Watcher

Independently verifies the active phase according to `WATCHER.md`.

### Archivist / Scribe

Promotes only durable execution knowledge into project artifacts.

## Information Management

Use `decisions/` for durable architectural/product decisions, `docs/` for durable technical knowledge, the active task file for execution evidence, and `reports/` for transient review/verification output.

If forgetting something could cause a future agent to make the wrong implementation choice, record it in the correct durable artifact. Otherwise, don't.
