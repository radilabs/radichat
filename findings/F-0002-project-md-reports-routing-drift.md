# F-0002 — PROJECT.md still routes transient reports to `reports/` instead of `.factory/reports/`

- **ID / title:** F-0002 — `PROJECT.md` Information Management routing stale after Factory v1.2 migration
- **Status:** triaged
- **Source:** Coder (T3 thread `46c38865-da1d-4849-a2da-65f754043115`, factory_droid), surfaced during the owner-authorized context-refresh run completed 2026-10-13T09:11:04Z; independently verified against the repository the same day
- **Date:** 2026-10-13
- **Project phase:** Stage 2 / Phase 3 — Factory v1.2 Migration (accepted, checkpoint commit `270108a`)

## Expected

Under Factory Core Concepts v1.2 (decision `0002-factory-v1.2-migration.md`, item 3), runtime reviewer/Watcher reports are written under the ephemeral `.factory/reports/`; durable defects go to `findings/`. `PROJECT.md`'s Information Management section should name the same routing, consistent with `AGENTS.md` and `tasks/README.md`.

## Observed

`PROJECT.md:166` still reads: "Use `decisions/` for durable architectural/product decisions, `docs/` for durable technical knowledge, the active task file for execution evidence, and `reports/` for transient review/verification output." The `reports/` location is the pre-v1.2 routing; `AGENTS.md` and `tasks/README.md` already point at `.factory/reports/`, so this one paragraph is the stale outlier.

## Reproduction

```sh
grep -n 'reports/' PROJECT.md          # line 166 names reports/ for transient output
grep -n 'factory/reports' tasks/README.md decisions/0002-factory-v1.2-migration.md
```

## Evidence

- `PROJECT.md:166` vs `decisions/0002-factory-v1.2-migration.md` (item 3), `tasks/README.md` operating rules ("Runtime reviewer and Watcher reports go under `.factory/reports/` (ephemeral, ignored); durable defects go under `findings/`"), and `.factory/reports/README.md`.

## Impact / severity

Minor — documentation drift. Could mislead a future agent into writing runtime reports into the root `reports/` tree (which is gitignored and, per the publication rule, must never be tracked) instead of `.factory/reports/`. No current behavioral impact.

## Triage result

Confirmed documentation drift (text stale), not implementation drift. Not a duplicate.

## Linked tasks / decisions

- `decisions/0002-factory-v1.2-migration.md` (item 3), `tasks/README.md`, Stage 2 / Phase 3.

## Closure

Disposition: fix deferred by the owner on 2026-10-13 to a future release — not converted to a task in this pass. Planned correction (smallest): reword the `PROJECT.md` sentence to route transient review/verification output to `.factory/reports/`, noting that the historical root `reports/` tree remains local-only evidence for accepted phases. No regression protection needed. Open.
