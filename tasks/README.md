# Task Execution

Current phase: **Phase 0 — Factory Bootstrap (accepted)**

`tasks/phase-0.md` is the authorized task file, populated from the immutable Phase 0 contract in `PHASES.md`. Phase 0 passed independent verification and is accepted. No later phase is authorized.

## Operating Rules

* Read `PROJECT.md`.
* Read `PHASES.md`.
* Read `TASKS.md`.
* Read only the currently authorized phase task file, if any.
* If none is authorized, STOP.
* Do not work outside the current phase.
* Discoveries outside scope go to **Deferred Work** in the current phase task file.
* A completed task does not imply a completed phase.
* Phase completion requires satisfying the current phase acceptance criteria and handoff contract in `PHASES.md`.
* Coder completion claims are not verification evidence.
* The Watcher must verify the actual repository state at the mandatory gates defined in `DRIVER.md` and `WATCHER.md`.
* Record decisions under `decisions/` only when future work must respect them.
* Record durable technical knowledge under `docs/` only when future work needs it.
* Keep the current phase task file updated with implementation progress, direct evidence, tests/results, files changed, known limitations, deferred work, relevant decisions, and handoff state.
* Runtime reviewer and Watcher reports go under `reports/` and are ignored by Git by default.
* Watcher PASS is necessary but does not complete the phase.
* Project-owner acceptance is required before commit/transition.
* Stop at phase handoff.
* Never create or begin the next phase task file automatically.

## Current Execution

Phase 0 is accepted following independent Watcher PASS (attempt 02) and the owner’s explicit conditional acceptance. See `docs/handoffs/phase-0.md`. Execution is stopped; Phase 1 requires separate authorization.
