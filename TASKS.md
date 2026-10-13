# RadiChat — Tasks

Detailed implementation tasks live under `tasks/`.

Only the currently authorized phase has a detailed task file that may be executed.

## Current Execution

- **Current stage:** Stage 2 — Factory v1.2 Runtime
- **Current phase:** Phase 3 — Factory v1.2 Migration
- **Status:** accepted

## Task File Map

```text
tasks/
├── README.md
├── phase-0.md
├── phase-1.md  (accepted)
├── phase-2.md  (accepted)
└── phase-3.md  (accepted)
```

Phase 3 is accepted. Phase 2 and Phase 1 remain accepted.

## Authorization

Owner explicitly authorized Phase 2 on 2026-09-22, with optional environment-sourced bearer authentication, strict TLS, no credential leakage, and an owner acceptance gate before any checkpoint commit or push.

Owner explicitly accepted Phase 2 on 2026-09-23 after successful endpoint testing and final independent Watcher PASS.

Owner explicitly authorized Phase 3 on 2026-10-13: migrate the repository process structure to Factory Core Concepts v1.2 (full conformance; Stage 2 / Phase 3 framing; historical `reports/` preserved local-only). Pre-migration baseline `bf60cc754f5ec0e47233d88839d3c0200993ee3f`, tagged `pre-factory-v1.2`.

Owner explicitly accepted Phase 3 on 2026-10-13 after independent Watcher PASS on AC1–AC8; checkpoint commit and push to `main` authorized with conditions: preserve the verified state; no product code changes; no scope expansion.

## History

Last completed stage: **Stage 2 — Factory v1.2 Runtime**.

Last completed phase: **Phase 3 — Factory v1.2 Migration**.

## Authority

The stage/phase roadmap and immutable contracts live in `PHASES.md`.

The product definition and factory rules live in `PROJECT.md`.

Driver behavior lives in `.factory/DRIVER.md` (materialized Factory kit).

Watcher verification behavior lives in `.factory/WATCHER.md` (materialized Factory kit).

Do not create or execute later-phase task files in advance.
