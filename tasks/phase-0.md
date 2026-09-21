# RadiChat — Stage 0 / Phase 0: Factory Bootstrap

## Goal

Populate the repository with the current Factory skeleton and establish RadiChat's initial product and phase contracts.

## Accepted Baseline

Commit: **695fe2e9d8c47d3a052c529d9df7a9d64a69baa0**

Initial repository state contained only the connector write probe.

This is the starting baseline, not the completed bootstrap checkpoint. The current pre-verification HEAD is `40fe0c38525ad5691bfb8f30be7343eb644bad1f`; the accepted Phase 0 checkpoint will follow verification and acceptance.

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

- [x] Inspect final repository tree.
- [x] Verify no product implementation exists.
- [x] Obtain independent Watcher PASS (attempt 02).
- [x] Owner explicitly instructed completion if verification is clean; condition satisfied by attempt 02 PASS.

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

Repository tree verified at `b354ccf2ed65b55390822ff544f3bde163db518a`: Factory files are present, the connector probe is gone, no Go source or Phase 1 task file exists. Watcher remains pending.

## Files Changed

Bootstrap files under repository root plus `tasks/`, `docs/`, `decisions/`, and `reports/`.

## Watcher / Review Status

- Dr Watson report(s): none
- Watcher attempt(s): `reports/stage-0-phase-0-watcher-01.md` — FAIL; findings under evidence-based recheck.
- Latest Watcher result: **PASS — attempt 02**, `reports/stage-0-phase-0-watcher-02.md`. All five acceptance criteria and handoff readiness independently verified; unsupported attempt 01 findings retracted.
- Transport: the local CAO service refused connection outside the sandbox. Used the permitted direct Droid CLI fallback with model `custom:Step-3.7-Flash-0`, `--auto low`, and only verification/report-writing authority. The local session record resolved a worker-reported identifier mismatch. Attempt 1 exited with code 0 but returned FAIL; process completion did not satisfy the gate.
- Owner confirmed Phase 0 is verifying and assigned Planner / Driver roles. Instruction to mark Phase 0 complete "if all well" is conditional acceptance effective only upon independent Watcher PASS.

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

**ACCEPTED — 2026-09-20**

Independent Watcher PASS satisfied the owner’s explicit conditional acceptance. Planner / Driver recorded acceptance and created `docs/handoffs/phase-0.md`. Stage 0 bootstrap is complete; no Phase 1 work is authorized.

Then STOP.

Do not begin Phase 1 until it is explicitly authorized.

## Final Verification and Transport Evidence

- Attempt 01 continuation in its persisted session exited 1 without output or new report. One bounded replacement used the same Watcher role and `custom:Step-3.7-Flash-0` model.
- Replacement Droid session: `b8c94c60-5970-49d4-b3ed-c25205716869`; explicit report result: PASS.
- Direct checks: `git ls-files`, filesystem inspection, `git status`, `git diff`, and `git diff --check`. No software build/test applies to the documentation-only bootstrap.
- Final lifecycle files changed: `TASKS.md`, `tasks/README.md`, `tasks/phase-0.md`, `README.md`, and `docs/handoffs/phase-0.md`. Raw Watcher reports remain ignored.
- Stage limitation: no separate Stage 0 exit conditions are enumerated. Phase 0 acceptance/handoff requirements serve as the bootstrap gate; future stage contracts should explicitly enumerate their exit conditions.
- Dr Watson was not invoked: this documentation-only closure introduced no implementation risk requiring separate review.
