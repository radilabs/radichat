# F-0001 — Phase 3 task file status field says "in progress" after acceptance

- **ID / title:** F-0001 — `tasks/phase-3.md` status field drift after Phase 3 acceptance
- **Status:** triaged
- **Source:** independent Watcher (T3 thread `c735bfd9-75d3-42a9-9927-82e1dc9a97aa`, poolside), surfaced during the owner-authorized context-refresh run completed 2026-10-13T09:12:47Z; independently verified against the repository the same day
- **Date:** 2026-10-13
- **Project phase:** Stage 2 / Phase 3 — Factory v1.2 Migration (accepted, checkpoint commit `270108a`)

## Expected

After owner acceptance, every lifecycle-state surface reports Phase 3 as accepted: `TASKS.md` (`Status: accepted`), `tasks/README.md` (`Current phase: Phase 3 … (accepted)`), and the task file's own Handoff Status section (`ACCEPTED — CHECKPOINT COMMIT/PUSH AUTHORIZED`). The task file's top-level status field should agree.

## Observed

`tasks/phase-3.md:5` still reads `Status: **in progress**.` while the same file's Handoff Status section, `TASKS.md`, and `tasks/README.md` all record the phase as accepted on 2026-10-13 (HEAD `270108a`).

## Reproduction

```sh
grep -n 'Status:' tasks/phase-3.md     # line 5: "Status: **in progress**."
grep -n 'Status' TASKS.md              # "Status: accepted"
```

## Evidence

- `tasks/phase-3.md:5` vs `tasks/phase-3.md` Handoff Status section, `TASKS.md` Current Execution, `tasks/README.md` header and Current Execution.
- Checkpoint commit `270108a` ("process: migrate repository to Factory v1.2 …") on `main`, pushed to `origin/main`.

## Impact / severity

Minor — lifecycle-state inconsistency only. No behavioral, product, or acceptance-criteria impact; the accepted evidence itself is intact. Risk is a future agent misreading the phase as still open.

## Triage result

Confirmed documentation drift (text stale), not implementation drift. Not a duplicate. No uncertainty material to the finding.

## Linked tasks / decisions

- `tasks/phase-3.md` (Stage 2 / Phase 3), `decisions/0002-factory-v1.2-migration.md`.

## Closure

Disposition: fix deferred by the owner on 2026-10-13 to a future release — not converted to a task in this pass. Planned correction (smallest): set the `tasks/phase-3.md` status line to `accepted`, matching `TASKS.md` and the handoff. No regression protection needed beyond the consistency itself. Open.
