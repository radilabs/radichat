# Phase Handoffs

`docs/handoffs/` stores concise, committed snapshots of accepted phase state.

A handoff is created only after a phase has been explicitly accepted by the Roboticist + Planner.

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

Keep handoffs concise. Promote only durable accepted state. Raw Watcher and Dr Watson reports remain under ignored `reports/`.
