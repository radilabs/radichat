# Phase Handoffs

`docs/handoffs/` stores concise, committed snapshots of accepted phase state, and durable release records for published releases.

A phase handoff is created only after a phase has been explicitly accepted by the Roboticist + Planner.

## Index

- [Phase 0 handoff](phase-0.md) — Factory bootstrap, accepted 2026-09-20.
- [Phase 1 handoff](phase-1.md) — local chat core, accepted 2026-09-21.
- [Phase 2 handoff](phase-2.md) — endpoint authentication, accepted 2026-09-23.
- [Phase 3 handoff](phase-3.md) — Factory v1.2 migration, accepted 2026-10-13.
- [Phase 4 handoff](phase-4.md) — `radichat-mock`, accepted 2026-10-14.
- [Phase 5 handoff](phase-5.md) — adversarial protocol investigation, accepted 2026-10-14.
- [Phase 6 handoff](phase-6.md) — v0.1.1 file I/O and batch support, accepted 2026-10-23.
- [v0.1.1 release record](release-v0.1.1.md) — published tag, GitHub release, GHCR image digest, verification, and limits. Release record, not a phase handoff. Published 2026-10-24.

It is not a raw reviewer report and it is not an execution journal.

## Purpose

The handoff lets future agents answer:

- what state was accepted at the end of the phase;
- what changed materially;
- what tests/evidence passed;
- what remains known or deferred;
- what baseline the next phase inherits.

## Naming

Use:

`phase-N.md`

## Required Structure

```markdown
# Phase N handoff

Date: YYYY-MM-DD
Accepted: YYYY-MM-DD
Status: **accepted**

## Stage / Phase
- Stage: ...
- Phase: ...

## Accepted Baseline
- Commit: ...
- Version/release state: ...

## Material Outcomes
- ...

## Findings Addressed
- ...

## Findings Deferred
- ...

## Tests / Evidence
- command/check — PASS/FAIL

## Owner Validation
- ...

## Known Limitations
- ...

## Notes for Next Phase
- ...
```

Keep handoffs concise. Promote only durable accepted state. Raw Watcher and Dr Watson reports remain under ignored `reports/`; handoffs summarize accepted outcomes so readers do not need those runtime files.
