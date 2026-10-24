# Task Execution

Current phase: **None authorized.** Stage 4 / Phase 6 is accepted and closed.

Phases 0–6 are accepted. Phase 4's durable handoff is `docs/handoffs/phase-4.md`. Phase 5 was accepted by the Owner on 2026-10-14 after independent Watcher PASS. Its durable handoff is `docs/handoffs/phase-5.md`; reproducible test summaries are in `docs/testing-phase-5.md`. Stage 4 / Phase 6 was accepted and closed on 2026-10-23; its task file is historical: `tasks/phase-6.md`.

The Owner separately authorized local v0.1.1 OCI packaging preparation on 2026-10-24. This bounded release task is recorded in `tasks/release-oci-v0.1.1.md`; it is not a phase and does not authorize product changes, a tag, release, or publication.

## Operating Rules

* Read `PROJECT.md`.
* Read `PHASES.md`.
* Read `TASKS.md`.
* Read the currently authorized phase task file, if any. For separately Owner-authorized non-phase release preparation, read only its named task record.
* If neither a phase nor a bounded non-phase task is explicitly authorized, STOP.
* Do not work outside the current authorization boundary.
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

Owner approved the v0.1.1 design at commit `334e9710efeab981782870df4bc0e050ef195be7` and authorized Stage 4 / Phase 6 on 2026-10-23. The accepted task is `tasks/phase-6.md`. Merge, tag, release, and publication were not authorized by that phase.

Owner authorized v0.1.1 local OCI packaging preparation on 2026-10-24. The separate bounded task is `tasks/release-oci-v0.1.1.md`. It authorizes packaging implementation, local builds, verification, documentation corrections, and necessary preparation commits only; it does not reopen Phase 6 or authorize tagging, release, GHCR push, or publication.
