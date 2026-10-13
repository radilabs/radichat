# Phase 3 handoff

Date: 2026-10-13
Accepted: 2026-10-13
Status: **accepted**

## Stage / Phase

Stage 2 — Factory v1.2 Runtime / Phase 3 — Factory v1.2 Migration.

## Accepted Baseline

The checkpoint commit containing this handoff is the accepted Phase 3 and Stage 2 baseline. Pre-migration baseline: `bf60cc754f5ec0e47233d88839d3c0200993ee3f`, tagged `pre-factory-v1.2`. Core Concepts pin: `12d4682c9e6497d03a0ba5a94652a6fc200ab1af`.

## Material Outcomes

The repository runs Factory Core Concepts v1.2 with full persona coverage. `AGENTS.md` routes Driver, Architect, Coder Team, Writer/Documentation Steward, and Watcher to materialized `.factory/` entrypoints and retains the CAO delegated-transport rules; `PROJECT.md` records the Architect and Writer roles.

Factory execution machinery is ephemeral: the `.factory/` kit (role entrypoints, canonical contract snapshots, defaults, pipeline) is materialized from the pinned Core Concepts commit, is gitignored, and is disposable. Role entrypoints are no longer project source; root `DRIVER.md`, `WATCHER.md`, and `reports/README.md` are untracked, and the public-surface check passes.

`decisions/0002-factory-v1.2-migration.md` records the pin rule (exact commit SHA, never a moving branch), the two-surface model, and the publication rule. `findings/` is the durable defect inbox; runtime reports route to `.factory/reports/`. Historical `reports/` contents are preserved local-only.

## Findings Addressed

The pre-migration audit found: the Architect persona absent from the project's role coverage; the Writer role stale at v1.0 wording; and a publication violation — tracked `DRIVER.md`, `WATCHER.md`, and `reports/README.md` in a public repository. Independent Watcher verification passed all eight acceptance criteria and additionally proved kit provenance by re-materializing from the pinned SHA and diffing (byte-identical except the project-filled `pipeline.yaml`).

## Tests / Evidence

- `sh .factory/check-public-surface.sh` — PASS (exit 1 before migration).
- `git ls-files` — no `.factory/`, `reports/`, root `DRIVER.md`/`WATCHER.md` paths.
- `.factory/source.txt` matches `^core_concepts_commit: [0-9a-f]{40}$` and equals the SHA recorded in `decisions/0002`.
- Kit re-materialized from the pin into a scratch directory — byte-identical except the project-filled `pipeline.yaml`.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` — PASS (5 packages); `go run . -help` — PASS.
- `git diff pre-factory-v1.2 -- '*.go' go.mod` — empty.
- `git diff pre-factory-v1.2` on accepted tasks, handoffs, and `decisions/0001` — empty; `PHASES.md` — 75 insertions, 0 deletions.
- Independent Watcher PASS: `.factory/reports/stage-2-phase-3-watcher-01.md` (ephemeral).

## Owner Validation

The owner resolved the migration framing decisions, explicitly authorized Phase 3 on 2026-10-13, and accepted it after independent Watcher PASS, authorizing the checkpoint commit and push to `main`.

## Known Limitations

- `.factory/` is disposable; a fresh clone must re-materialize from the pinned Core Concepts SHA before delegated work.
- `.factory/integration-plan.yaml` retains template placeholders; fill it only if parallel-writing work-integration flows are used.
- The pipeline bootstrap command assumes `core-concepts` is a sibling directory of the repository root.
- Historical `reports/` contents remain local-only evidence, as before the migration.

## Deferred Work

Promotion of selected historical report evidence into `findings/`/`docs/`, any product feature work, and any Factory version upgrade beyond pin `12d4682c…` remain outside the accepted scope and require explicit authorization.

## Notes for Next Phase

Stage 2 is complete. No later phase is authorized. Preserve the exact-SHA pin rule, the two-surface persistent/ephemeral split, the `findings/` inbox, and the publication-boundary check before any release or container publication in any future work.
