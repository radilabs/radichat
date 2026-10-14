# RadiChat — Stage 3 / Phase 4: `radichat-mock`

## Authorization and Status

Status: **accepted** (Owner acceptance: 2026-10-14).

Owner explicitly authorized Stage 3 / Phase 4 on 2026-10-14 after approving the Stage 3 placement, loopback-only development CLI, separate Phase 5 authorization gate, exclusion of product fixes, repository-based showcase evidence, and fixed October 15, 2026 23:59 UTC deadline.

The immutable contract is `PHASES.md` → Stage 3 / Phase 4. This task decomposes that contract and does not expand it.

## Contract Summary

Build the standard-library Go `internal/mockllm` package and loopback-only `cmd/radichat-mock` development executable. Provide deterministic named OpenAI-compatible scenarios, safe request recording, focused tests, and usage documentation. Do not modify RadiChat non-test product code, run the full adversarial matrix against RadiChat, record defect findings, publish the showcase, or start Phase 5. Phase 5 and any product fixes require separate Owner authorization.

## Baseline

- Authorized repository baseline: `f153edf` (main, one commit ahead of `origin/main`).
- Phase 3 accepted; F-0001 and F-0002 are deferred and remain outside this phase.
- Existing protocol oracle: `docs/protocol.md`.
- Existing client/context behavior references: `internal/client/`, `internal/chat/`, `internal/repl/`, `docs/configuration.md`, and Phase 1/2 handoffs.
- Deadline: **2026-10-15 23:59 UTC**. Intermediate targets from the proposal are advisory.

## Implementation Plan

### 4.0 Contract and baseline

- [x] Record the explicit Owner authorization and exact phase boundary.
- [x] Confirm baseline before implementation: only the authorized Phase 4 contract/state files are modified; accepted Phase 0–3 task files, handoffs, and decisions are unchanged; `git diff --check` passes.
- [x] Record the exact candidate identity before Watcher verification.

### 4.1 Mock package and scenario catalog

- [x] Implement `internal/mockllm/` using only the Go standard library.
- [x] Add deterministic named scenarios for successful streams; SSE framing and fragmented bytes; malformed and incomplete responses; HTTP errors/non-SSE responses; bearer auth match/missing/mismatch; redirects; and context-accounting request fixtures.
- [x] Add deterministic fragment helpers with fixed offsets or a recorded seed. Test the configured byte segments without assuming the network preserves write boundaries.
- [x] Record request method, path, relevant non-secret headers, decoded request fields, and auth outcome; never store or emit bearer-token values.
- [x] Test loopback-only startup defaults, scenario behavior, redaction, and clean shutdown.

### 4.2 Development CLI and documentation

- [x] Implement `cmd/radichat-mock` as a thin wrapper to list/run named scenarios, bind only to loopback, print its endpoint, and shut down on interrupt.
- [x] Verify the root build remains the RadiChat executable and inspect repository packaging definitions for mock inclusion. No release/container build definitions are present to inspect, so exclusion from packaging systems outside this repository cannot be demonstrated here.
- [x] Write `docs/testing-mock.md` with reproducible commands and explicit Phase 5 boundary.

### 4.3 Coder verification and handoff

- [x] Run `gofmt` on changed Go files.
- [x] Run `go test ./... -count=1`, `go vet ./...`, `go build ./...`, `go run . -help`, and `sh .factory/check-public-surface.sh`.
- [x] Verify no third-party module was added, no non-test product-code file changed, and accepted Phase 0–3 records/evidence are preserved.
- [x] Return changed-file list, candidate identity, exact command results, limitations, and deferred work.

### 4.4 Independent Watcher and owner gate

- [x] Freeze the candidate; make no edits while the Watcher verifies it.
- [x] Provide the Watcher the immutable contract, this task, candidate identity, diff, and direct test/build evidence.
- [x] Watcher independently verifies each acceptance criterion and returns explicit PASS or FAIL with evidence in `.factory/reports/`.
- [x] Correct only in-scope issues, rerun checks, freeze a new candidate, and obtain Watcher re-verification. Not required: Watcher returned PASS; no correction/reverification cycle was needed.
- [x] Assemble the evidence package and STOP for Owner acceptance. No commit, phase transition, Phase 5 work, product fix, or showcase publication before the applicable authorization.

## Verification Matrix

| Contract criterion | Required direct evidence |
| --- | --- |
| Standard-library mock; loopback; shutdown | `go list -deps` / source inspection; mock tests and runtime shutdown evidence |
| Named scenario coverage | Catalog listing and package tests for each required scenario family |
| Exact deterministic fragmentation | Tests assert configured segment boundaries; no assertion that TCP preserves individual writes |
| Safe request recorder | Tests inspect recorded fields and assert sentinel token absent from recorder, stdout, stderr, and reports |
| CLI and distribution boundary | CLI list/run evidence; root build output; inspect release/container definitions for exclusion |
| Reproducible instructions | `docs/testing-mock.md` commands executed and recorded |
| Checks and source boundary | Exact outputs for test, vet, build, help, public-surface; diff confirms no non-test product code or accepted record changed |
| Independent verification | Watcher report identifies candidate and AC-by-AC PASS/FAIL evidence |

## Authorization Boundaries

### Included now: Phase 4

Implement and validate the mock package, scenario catalog, CLI, tests, and documentation. Tests may validate the mock itself and a known-supported RadiChat smoke path. Merely defining adversarial scenarios and request fixtures does not authorize running the full matrix against RadiChat.

### Not authorized: Phase 5

Adversarial black-box execution against RadiChat, defect triage, and durable `findings/` records are deferred until separate Owner authorization after Phase 4 acceptance. A no-defect result is acceptable; no minimum bug count applies.

### Not authorized: product fixes or publication

No product-code fix, release/container integration, showcase publication, or external communication is authorized by this phase.

## Risks / Limitations

- HTTP servers and TCP may coalesce writes; the contract verifies deterministic segment generation without promising socket-read boundaries.
- The mock exercises the documented OpenAI-compatible subset; it is not a general-purpose model server.
- Context accounting fixtures assert the documented byte-based policy and must not claim to measure model tokens.
- Mock infrastructure alone cannot establish whether RadiChat has a defect; that investigation is separately gated.

## Deferred Work

- Full adversarial bug hunt and reproducible findings: Phase 5, separate authorization required.
- Owner-selected RadiChat fixes: separate phase/authorization after finding review.
- Human visual demonstration or showcase publication: not required or authorized.

## Evidence and Handoff Status

### Candidate and changed files

Base: `f153edf` on `main`. The accepted checkpoint is the commit containing `docs/handoffs/phase-4.md`; it preserves the Watcher-verified candidate and records acceptance state.

- `PHASES.md`, `TASKS.md`, `tasks/README.md`, `tasks/phase-4.md` — authorized Stage 3 / Phase 4 authority and execution state.
- `internal/mockllm/` — standard-library server, catalog, recorder, fragmentation support, and tests.
- `cmd/radichat-mock/` — development-only CLI and tests.
- `docs/testing-mock.md`, `docs/README.md` — usage, privacy limits, and index entry.

Candidate identity (initial implementation candidate verified by Watcher): base `f153edf` plus SHA-256 `6b30033f0b9e51d610372953dd11d6fa57c5733d8387169ccd4e82f6a4ea4222` over each changed file except this task file (sorted paths; hash input is path, NUL, raw per-file SHA-256, newline). The current `tasks/phase-4.md` is included separately in the Watcher handoff.

Updated documentation candidate after the requested TASKS.md cleanup: base `f153edf` plus SHA-256 `3eeb6ecbd04a76d2e22fafcd88cad56f09489e327ef779d4086df009689cd097` over each changed file except this task file (same algorithm). The task file itself is included separately in the follow-up Watcher handoff. No implementation files changed.

### Driver verification

- `go test ./... -count=1` — PASS, 7 packages.
- `go test ./... -count=1 -race -timeout 300s` — PASS, 7 packages.
- `go vet ./...` — PASS.
- `go build ./...` — PASS.
- `go run . -help` — PASS.
- `sh .factory/check-public-surface.sh` — PASS: no configured Factory runtime paths are tracked.
- `git diff --check` — PASS.
- `git diff HEAD -- '*.go' go.mod go.sum` — empty; no product-code or module manifest changes.
- `go list -deps` for the mock package and CLI reports only the project package paths as non-standard-library dependencies.
- `find` found no Dockerfile, Containerfile, release definition, or GitHub workflow at the inspected repository depth. The root command remains RadiChat; packaging exclusion cannot be verified because no packaging definitions exist in this repository.
- The Coder added regression tests for Cookie omission and query-string omission. Its query regression test was verified by temporarily restoring query capture (the test failed with the sentinel present), then restoring the fix. These tests are included in the passing suite.
- The full adversarial matrix against RadiChat was not run. Phase 5 remains unauthorized.

### Handoff status

Independent Watcher: PASS on all eight acceptance criteria (2026-10-14). Initial report: `.factory/reports/stage-3-phase-4-watcher-01.md`; documentation/state re-verification: `.factory/reports/stage-3-phase-4-watcher-02.md`. Both reports are ephemeral runtime evidence; the durable handoff summarizes their results.

Owner acceptance: **accepted 2026-10-14** after review of the evidence package and Watcher PASS. Phase 4 is complete. No Phase 5 work, product-code changes, or showcase publication is authorized; stop here pending separate authorization.
