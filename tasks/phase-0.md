# RadiChat — Stage 0 / Phase 0: Factory Bootstrap

## Goal

Populate the repository with the current Factory skeleton and establish RadiChat's initial product and phase contracts.

## Accepted Baseline

Commit: **695fe2e9d8c47d3a052c529d9df7a9d64a69baa0**

Initial repository state contained only the connector write probe.

## Read First

1. `PROJECT.md`
2. `PHASES.md`
3. `TASKS.md`
4. `tasks/README.md`
5. this task file

## Entry Conditions

* Repository exists.
* Owner explicitly requested Factory bootstrap.
* This task file reflects the Phase 0 contract.
* No implementation is required.

## Contract Reference

Stage: **0 — Bootstrap**

Phase contract: **`PHASES.md` → Phase 0 — Factory Bootstrap**

## Scope

* Factory skeleton files.
* Product definition.
* Stage/phase contracts.
* execution index and Phase 0 task file.
* project-specific ignore rules.

## Explicit Exclusions

* Go implementation.
* Phase 1 task file.
* Authentication.
* Persistence/database/compaction.

## Tasks

### Task 0.1 — Install Factory structure

- [x] Copy role/lifecycle skeleton files.
- [x] Create standard docs, decisions, reports, and tasks structure.
- [x] Add project-specific ignore rules.

### Task 0.2 — Define RadiChat

- [x] Record product principles and non-goals.
- [x] Define Phase 1 unauthenticated Linux single-binary client.
- [x] Define Phase 2 authentication.
- [x] Defer persistence/database/compaction.

### Task 0.3 — Verify bootstrap

- [ ] Inspect final repository tree.
- [ ] Verify no product implementation exists.
- [ ] Obtain independent Watcher PASS.
- [ ] Present to owner for acceptance.

## Acceptance Criteria

1. Required Factory skeleton files exist at repository root and standard directories exist.
2. `PROJECT.md` describes the agreed product and non-goals.
3. `PHASES.md` defines the agreed Phase 1 and Phase 2 boundary.
4. `TASKS.md` reports Phase 0 as the only authorized detailed task.
5. No implementation or future phase task file exists.

## Verification Plan

- **AC1:** repository tree inspection.
- **AC2:** direct `PROJECT.md` inspection.
- **AC3:** direct `PHASES.md` inspection.
- **AC4:** direct `TASKS.md` plus task tree inspection.
- **AC5:** repository tree inspection.

## Implementation Progress

Factory skeleton copied and RadiChat contracts written.

## Tests and Evidence

Pending final repository-tree verification and Watcher.

## Files Changed

Bootstrap files under repository root plus `tasks/`, `docs/`, `decisions/`, and `reports/`.

## Watcher / Review Status

- Dr Watson report(s): none
- Watcher attempt(s): none
- Latest Watcher result: **NOT RUN**

## Known Limitations

- No RadiChat executable exists yet by design.
- Configuration file syntax is deliberately not frozen in Phase 0; Phase 1 may choose the smallest sensible format while satisfying its contract.

## Deferred Work

- Phase 1 implementation.
- Phase 2 authentication.
- Later optional session persistence/database/compaction.

## Regression Memory

- none; no implementation exists.

## Decisions / Durable Knowledge

- Product and phase boundaries are recorded in `PROJECT.md` and `PHASES.md`.

## Handoff Status

**NOT READY**

Then STOP.

Do not begin Phase 1 until it is explicitly authorized.
