# Runtime Reports

`reports/` stores temporary execution reports produced by independent reviewers and verifiers during the active development loop.

Report contents are **not** a permanent source of truth and are ignored by Git by default.

If a report contains knowledge or a decision that future work must respect, deliberately promote that information into the appropriate durable artifact:

- `tasks/phase-N-*.md` for phase execution evidence and temporary findings that matter to the handoff;
- `docs/` for durable technical knowledge;
- `decisions/` for durable decisions.

Do not commit raw reports merely because they exist.

## Watcher Reports

Recommended filename:

`stage-<S>-phase-<N>-watcher-<attempt>.md`

Example:

`stage-0-phase-2-watcher-03.md`

Watcher reports follow `WATCHER.md`.

## Dr Watson Reports

Recommended filename:

`stage-<S>-phase-<N>-watson-<attempt>.md`

Example:

`stage-0-phase-2-watson-01.md`

A Dr Watson report should include:

- scope reviewed;
- evidence inspected;
- ranked findings;
- why each finding matters;
- whether each finding is in-scope or recommended Deferred Work;
- unresolved questions.

## Rule

Reports support the loop. Durable project truth lives elsewhere.
