# RadiChat — Stage 2 / Phase 3: Factory v1.2 Migration

## Authorization and Status

Status: **in progress**.

Owner explicitly authorized Phase 3 on 2026-10-13 after resolving the migration framing decisions (full v1.2 conformance — no partial migration; new Stage 2 / Phase 3; historical `reports/` kept local-only with contents preserved; this plan promoted to `tasks/phase-3.md` after authorization). Pre-migration baseline: `bf60cc754f5ec0e47233d88839d3c0200993ee3f`, tagged `pre-factory-v1.2`.

Execution plan source: `.hermes/plans/2026-10-13_101233-radichat-factory-v1.2-migration.md` (Hermes workspace, untracked).

## Immutable Contract Reference

Authority: `PHASES.md` → Stage 2 / Phase 3. This section restates the contract without amendment.

### Goal

Migrate repository process structure to Factory Core Concepts v1.2 without changing product behavior or accepted phase evidence.

### Scope

- Persistent-surface updates: `AGENTS.md`, `PROJECT.md` roles, `findings/`, `tasks/README.md` alignment, `.gitignore`.
- Untrack Factory runtime paths (`DRIVER.md`, `WATCHER.md`, `reports/README.md`).
- Materialize the ephemeral `.factory/` kit from a pinned Core Concepts commit and fill project-specific pipeline commands.
- Record the migration as a durable decision.

### Explicit Exclusions

- Go source code, product behavior, configuration formats.
- `PHASES.md` contracts of accepted phases (Phase 0/1/2 remain immutable).
- Historical `reports/` contents.
- New product features or future phase task files.

### Acceptance Criteria

1. `AGENTS.md` routes all v1.2 managed roles (Driver, Architect, Coder, Writer, Watcher) to `.factory/` entrypoints and retains the CAO transport section.
2. `PROJECT.md` roles include Architect and Writer/Documentation Steward.
3. `git ls-files` contains no `.factory/`, `reports/`, root `DRIVER.md`, or root `WATCHER.md` paths; `sh .factory/check-public-surface.sh` prints PASS.
4. `.factory/` contains role entrypoints, canonical contract snapshots, `defaults.yaml`, `agents.yaml`, filled `pipeline.yaml`, and `source.txt`.
5. The Core Concepts pin is an exact commit SHA (40 hex chars — never a branch, tag, or moving ref), recorded in `.factory/source.txt` **and** durably in `decisions/0002-factory-v1.2-migration.md`.
6. `decisions/` records the migration decision; `findings/README.md` exists.
7. `go build ./...`, `go vet ./...`, and `go test ./...` pass unchanged.
8. Accepted phase task files, handoffs, and decisions are unmodified.

### Handoff Contract

Before Phase 3 can be declared complete: all acceptance criteria demonstrated with direct evidence; changed files and checks recorded; known limitations and deferred work recorded; Watcher verification PASS; owner explicitly accepts. Then STOP.

## Authorized Tasks

### 3.0 Authorization and baseline

- [x] Owner authorizes Phase 3 with resolved framing decisions (2026-10-13).
- [x] Append Stage 2 / Phase 3 contract to `PHASES.md` (75 insertions, 0 deletions; accepted contracts byte-identical).
- [x] Tag `pre-factory-v1.2` @ `bf60cc7`; working tree clean except untracked `.hermes/`.
- [x] Create this task file only after authorization.

### 3.1 Persistent surface

- [x] Rewrite `AGENTS.md` role routing for v1.2: `.factory/` entrypoint table (Driver/Architect/Coder/Writer/Watcher), Architect and Writer sections; retain the CAO transport section and Stop Rule.
- [x] Update `PROJECT.md` roles: add Architect, rename Archivist/Scribe → Writer/Documentation Steward, repoint Driver/Watcher to `.factory/` entrypoints.
- [x] Add `findings/README.md` (skeleton copy — durable defect inbox).
- [x] Align `tasks/README.md`: findings pointer + `.factory/reports/` routing (plus dangling `DRIVER.md`/`WATCHER.md` reference fix and live Phase 3 state; `TASKS.md` updated to in-progress state the same way).
- [x] `.gitignore`: remove `!reports/README.md` negation; add `.hermes/`.
- [x] Write `decisions/0002-factory-v1.2-migration.md` (pin rule + two-surface model + publication rule).

### 3.2 Publication boundary

- [x] `git rm --cached DRIVER.md WATCHER.md reports/README.md` (working files preserved until kit verified).
- [x] Remove root `DRIVER.md`/`WATCHER.md` working copies after kit verification (recoverable from `pre-factory-v1.2`).

### 3.3 Ephemeral Factory kit

- [x] Resolve the exact Core Concepts pin SHA: `12d4682c9e6497d03a0ba5a94652a6fc200ab1af` ("Wire v1.2 role contracts into Factory materialization", clean tree).
- [x] Materialize: `sh <core-concepts>/skeleton/materialize.sh . runtime`; entrypoints, contracts, defaults, `agents.yaml` architect binding, and `source.txt` 40-hex format all verified.
- [x] Fill `.factory/pipeline.yaml` with project-specific commands (bootstrap/build/vet/test/smoke/broad/public-surface/cleanup; deploy explicitly omitted; `pinned_source` = the exact SHA).
- [x] `sh .factory/check-public-surface.sh` prints **PASS** (exit 1 at baseline).
- [x] `go build ./... && go vet ./... && go test ./... -count=1` pass; `*.go`/`go.mod` diff vs baseline empty.

### 3.4 Independent Watcher and owner gate

- [x] Independent Watcher verifies AC1–AC8 from actual repository state (fresh, isolated context; no Driver substitution); report at `.factory/reports/stage-2-phase-3-watcher-01.md` — **PASS** (2026-10-13): AC1–AC8 by direct observation; kit provenance proven by re-materialization diff from the pinned SHA; dangling references adjudicated clean; smoke command verified. Notes N1–N3 recorded, none contract-bearing.
- [x] Assemble evidence, limitations/deferred work, and Watcher result for owner acceptance; stop at this gate.
- [x] After explicit owner acceptance: create `docs/handoffs/phase-3.md`, commit, update `TASKS.md`/`tasks/README.md`, STOP.

## Verification Matrix

| Criterion | Required direct evidence |
| --- | --- |
| AC1 | `AGENTS.md` routing table + Architect/Writer sections present; CAO transport section retained; `grep` evidence. |
| AC2 | `PROJECT.md` contains Architect and Writer/Documentation Steward roles. |
| AC3 | `git ls-files` free of `.factory/`, `reports/`, root `DRIVER.md`/`WATCHER.md`; `check-public-surface.sh` PASS. |
| AC4 | `.factory/` listing shows entrypoints + `contracts/` + `defaults.yaml` + `agents.yaml` + filled `pipeline.yaml` + `source.txt`. |
| AC5 | `source.txt` matches `^core_concepts_commit: [0-9a-f]{40}$`; same SHA in `decisions/0002`; no branch/tag used. |
| AC6 | `decisions/0002-*` and `findings/README.md` exist. |
| AC7 | `go build ./...`, `go vet ./...`, `go test ./... -count=1` all exit 0; empty `*.go`/`go.mod` diff vs `pre-factory-v1.2`. |
| AC8 | `git diff pre-factory-v1.2` touches no accepted task/handoff/decision content; `PHASES.md` diff is the Phase 3 append only. |

## Progress and Evidence

2026-10-13: Phase 3 authorized with resolved framing decisions. `PHASES.md` appended with the Stage 2 / Phase 3 contract (75 insertions, 0 deletions; lines 1–220 byte-identical to baseline). Baseline tag `pre-factory-v1.2` @ `bf60cc7`.

Implementation evidence (all commands run in the repository root):

- Persistent surface: `AGENTS.md` now routes Driver/Architect/Coder/Writer/Watcher to `.factory/` entrypoints and retains the CAO transport section; `PROJECT.md` roles gained Architect and Writer/Documentation Steward; `findings/README.md` added (identical to skeleton); `tasks/README.md` aligned (findings pointer, `.factory/reports/` routing, dangling root-contract references repointed); `TASKS.md` updated to Phase 3 state; `.gitignore` no longer negates `reports/README.md` and ignores `.hermes/`; `decisions/0002-factory-v1.2-migration.md` written.
- Publication boundary: `git rm --cached DRIVER.md WATCHER.md reports/README.md`; root `DRIVER.md`/`WATCHER.md` working copies removed after kit verification (recoverable from `pre-factory-v1.2`); `git ls-files` shows no forbidden paths.
- Ephemeral kit: materialized from Core Concepts pin `12d4682c9e6497d03a0ba5a94652a6fc200ab1af` (clean tree); `.factory/` contains DRIVER/WATCHER/ARCHITECT/CODER/WRITER entrypoints, 9 canonical contract snapshots, `defaults.yaml`, `agents.yaml` (architect → claude), filled `pipeline.yaml` (0 placeholders), `source.txt` (40-hex, matches decision 0002); `git check-ignore .factory/` succeeds.
- `sh .factory/check-public-surface.sh` → **PASS: no configured Factory runtime paths are tracked** (exit 1 at baseline).
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` → all pass (5 packages); `git diff --stat pre-factory-v1.2 -- '*.go' go.mod` → empty.

Independent Watcher verification completed 2026-10-13: **PASS** on AC1–AC8 (fresh isolated context, direct observation, no repairs). Report: `.factory/reports/stage-2-phase-3-watcher-01.md` (ephemeral). Owner accepted Phase 3 on 2026-10-13 with conditions: preserve the verified state; no product code changes; no scope expansion. Checkpoint commit and push to `main` authorized.

## Changed Files

Actual: modified `PHASES.md` (append only — 75 insertions, 0 deletions), `AGENTS.md`, `PROJECT.md`, `TASKS.md`, `tasks/README.md`, `.gitignore`; untracked/removed from index `DRIVER.md`, `WATCHER.md`, `reports/README.md` (root working copies of the first two deleted after kit verification); new `tasks/phase-3.md`, `findings/README.md`, `decisions/0002-factory-v1.2-migration.md`, `docs/handoffs/phase-3.md`. Ephemeral (gitignored): `.factory/**` including the Watcher report. No product files (`*.go`, `go.mod`, docs/, README) touched.

## Known Limitations / Deferred Work

- `.factory/` is disposable; a fresh clone must re-materialize from the pinned Core Concepts SHA recorded in `decisions/0002` before delegated work.
- Historical `reports/` contents remain local-only evidence (unchanged status quo); promotion of selected evidence into `findings/`/`docs/` is deferred to a future authorized phase if ever needed.
- The Core Concepts working tree should be clean at the pinned SHA; dirty state makes the pin non-reproducible (noted at materialization time).
- Watcher note N1: `.factory/integration-plan.yaml` retains unfilled `FILL_IN_*` placeholders — outside the Phase 3 contract (only `pipeline.yaml` required filling; the materializer ships this file as a template). Fill it if/when parallel-writing work-integration flows are first used.
- Watcher note N2: `pipeline.yaml` bootstrap command assumes `core-concepts` is a sibling directory of the repo root — true in this environment and documented in `decisions/0002`; re-record if the layout changes.

## Decisions / Regression Memory

- Core Concepts is pinned by exact commit SHA, never a moving branch (owner decision 2026-10-13).
- Role entrypoints live in the ephemeral kit; root `DRIVER.md`/`WATCHER.md` are not project source.
- Runtime reports route to `.factory/reports/`; durable defects route to `findings/`.
- Publication boundary: `.factory/`, `reports/`, root `DRIVER.md`/`WATCHER.md`, `factory-kit/` must never be tracked; `check-public-surface.sh` must pass before any release/container publication.

## Watcher / Review Status

Independent Watcher attempt 1 returned **PASS** on 2026-10-13 in `.factory/reports/stage-2-phase-3-watcher-01.md` (ephemeral). The Watcher ran with a fresh isolated context after the Driver reported implementation complete, executed every check directly against repository state, and performed no repairs. All AC1–AC8 verified; no blockers, no majors; notes N1–N3 are non-contract-bearing. Watcher note N3 confirms the uncommitted working tree is the contract-mandated pre-acceptance state. Phase 3 was presented at the owner-acceptance gate and explicitly accepted by the owner on 2026-10-13.

## Handoff Status

**ACCEPTED — CHECKPOINT COMMIT/PUSH AUTHORIZED.**

Owner acceptance recorded 2026-10-13 with conditions: preserve the verified state; no product code changes; no scope expansion. `docs/handoffs/phase-3.md` records the accepted Stage 2 checkpoint. No later phase is authorized.
