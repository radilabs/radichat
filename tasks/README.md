# Task Execution

Current phase: **Stage 4 / Phase 6 — v0.1.1 File I/O and Batch Support**

Phases 0–4 are accepted. Phase 4's durable handoff is `docs/handoffs/phase-4.md`. Phase 5 was accepted by the Owner on 2026-10-14 after independent Watcher PASS. Its durable handoff is `docs/handoffs/phase-5.md`; reproducible test summaries are in `docs/testing-phase-5.md`. Stage 4 / Phase 6 was authorized on 2026-10-23; its active task is `tasks/phase-6.md`.

## Operating Rules

* Read `PROJECT.md`.
* Read `PHASES.md`.
* Read `TASKS.md`.
* Read only the currently authorized phase task file, if any.
* If none is authorized, STOP.
* Do not work outside the current phase.
* Discoveries outside scope go to **Deferred Work** in the current phase task file; operator-observed defects also receive durable records under `findings/` for later triage. A finding is not permission to start work.
* A completed task does not imply a completed phase.
* Phase completion requires satisfying its acceptance criteria and handoff contract in `PHASES.md`, independent Watcher PASS, and explicit Owner acceptance.
* Coder completion claims are not verification evidence.
* The Watcher must verify the actual repository state at the mandatory gates defined in `.factory/DRIVER.md` and `.factory/WATCHER.md`.
* Record decisions under `decisions/` only when future work must respect them.
* Record durable technical knowledge under `docs/` only when future work needs it.
* Keep the current phase task file updated with implementation progress, direct evidence, tests/results, files changed, known limitations, deferred work, relevant decisions, and handoff state.
* Runtime reviewer and Watcher reports go under `.factory/reports/` (ephemeral, ignored); durable defects go under `findings/`.
* Watcher PASS is necessary but does not complete the phase.
* Project-owner acceptance is required before commit/transition.
* Stop at phase handoff.
* Never create or begin the next phase task file automatically.

## Authorization History

Owner accepted Phase 1 on 2026-09-21 and Phase 2 on 2026-09-23 after successful endpoint testing and independent Watcher PASS.

Owner authorized Stage 2 / Phase 3 (Factory v1.2 Migration) on 2026-10-13 and accepted it after independent Watcher PASS. That accepted checkpoint is preserved.

Owner authorized Stage 3 / Phase 4 (`radichat-mock`) on 2026-10-14. At the time, only `tasks/phase-4.md` was active; Phase 5, product-code fixes, and showcase publication were outside that authorization.

Owner accepted Stage 3 / Phase 4 on 2026-10-14 after independent Watcher PASS on AC1–AC8. Its accepted checkpoint is `8e2951274e52f72963ad8ecb1d75a4a477fe6561`.

Owner authorized Stage 3 / Phase 5 on 2026-10-14. It completed with zero confirmed divergences and independent Watcher PASS; Owner accepted it on 2026-10-14. Product-code fixes remain unauthorized; the Phase 5 contract's historical reference to Phase 6 meant the then-planned product-fix work. The showcase package is prepared for Owner review; publication still requires separate Owner authorization.

Owner approved the v0.1.1 design at commit `334e9710efeab981782870df4bc0e050ef195be7` and authorized Stage 4 / Phase 6 on 2026-10-23. The authorized task is `tasks/phase-6.md`. Merge, tag, release, and publication are not authorized.
