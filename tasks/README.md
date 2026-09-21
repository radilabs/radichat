# Task Execution

Current phase: **Phase 1 — Local Chat Core (accepted)**

`tasks/phase-1.md` records the accepted checkpoint. Phase 0 remains accepted; Phase 2 is not authorized. No phase is currently authorized for execution.

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

Owner accepted Phase 1 on 2026-09-21 after independent Watcher PASS. Stop at the accepted checkpoint. Do not create or begin Phase 2 without explicit authorization.
