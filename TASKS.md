# RadiChat — Tasks

## Current Execution

- **Completed stage:** Stage 4 — v0.1.1 File I/O and Batch Support
- **Current phase:** None authorized for execution.
- **Last accepted phase:** Stage 4 / Phase 6 — v0.1.1 File I/O and Batch Support
- **Status:** Phase 6 accepted by the Owner on 2026-10-23 after fresh independent Watcher PASS. Accepted implementation checkpoint `c77110d76633bef27dd1f4f347fe46c3c629d1ed` was pushed to `main`. No later phase is authorized.
- **Release status:** `v0.1.1` is published on 2026-10-24. Annotated tag `v0.1.1` peels to `6645988e354076fede28070686b23667d0ff89ac`. GitHub release: https://github.com/radilabs/radichat/releases/tag/v0.1.1. GHCR image `ghcr.io/radilabs/radichat:v0.1.1` is `sha256:10e843853f975998159e6110aeb55c5b9e1ec5db8edfd2f9d1ad59a9fc112505`. The image has not been run in a container. `latest` was not moved and still resolves to `v0.1.0`.
- **Release handoff:** `docs/handoffs/release-v0.1.1.md`.

Phases 0–6 are accepted. Phase 6 delivered v0.1.1 file I/O and batch support. Historical Phase 5 references to “Phase 6” meant the then-planned product-fix phase. No next phase is authorized. Any further tag, GitHub release, image publication, or `latest` change requires separate Owner authorization. Showcase publication also requires separate Owner authorization.

Accepted task records: `tasks/phase-5.md`, `tasks/phase-6.md`.
Owner-authorized release record: `tasks/release-oci-v0.1.1.md` (not a phase; publication completed 2026-10-24).
Durable Phase 6 handoff: `docs/handoffs/phase-6.md`.

Task map:

```text
tasks/phase-5.md  (accepted)
tasks/phase-6.md  (accepted; Stage 4 / Phase 6)
```

The accepted Stage 2 / Phase 3 checkpoint and all earlier accepted history remain preserved.

Detailed implementation tasks live under `tasks/`.

Phases 0–6 are accepted and preserved. No phase is currently authorized for execution.

## v0.1.1 Milestone Brief — File I/O and Batch Support (Owner-requested)

Owner requested this bounded next step on 2026-10-23. The brief below has been materialized as Stage 4 / Phase 6 in `PHASES.md`, and its active task file is `tasks/phase-6.md`. The design proposal was approved in commit `334e9710efeab981782870df4bc0e050ef195be7`. This authorization does not include merge, tag, release, or showcase publication. Existing accepted history remains unchanged.

**Product intent:** Keep RadiChat a small Unix-style OpenAI-compatible chat client; support manual local batch testing and execution inside a Kubernetes Job without coupling the client to Kubernetes.

**Requested behavior and scenarios:**

1. **System prompt from file:** Allow a readable UTF-8 file to supply the configured system message (proposed `system_prompt_file` JSON key). Mutually exclusive with inline `system_prompt`; fail clearly on conflicts, missing/unreadable/oversized files and when prompt cannot fit the context budget. Decide and document relative-path resolution.
2. **Local sequential batch:** Preserve existing stdin redirection (one input line = one user turn, sequentially, with existing conversation and `/clear` semantics). Provide opt-in machine-readable output, proposed `-output PATH` writing JSONL records that correlate input with successful complete assistant output. Define a precise policy for errors, partial streams, failed turns, file creation/appending/overwrite, interruption, and stdout behavior before coding. Never report partial output as a successful completion.
3. **Kubernetes Job example:** Document a Job using mounted config and prompt files (ConfigMap/volume as suitable), stdin or a narrowly scoped file-input option only if necessary, and stdout/stderr collection by Kubernetes. Output persistence, routing, and Job/CronJob scheduling belong to Kubernetes/logging infrastructure, not RadiChat. Container execution must remain non-interactive and work without a writable filesystem.
4. **Validation:** Add focused tests for configuration, file boundaries, successful and failed streamed turns, JSONL escaping/correlation, EOF, `/clear`, and backward compatibility; provide reproducible local and Kubernetes examples and run the repository's standard checks.

**Excluded:** database, session reload/resume, implicit transcript storage, concurrent requests, retries, scheduling/orchestration, Kubernetes client/API integration, agents/tools, new external dependencies, and changes to the existing supported inference protocol.

**Factory handoff:** Architect reviewed the exact CLI/config and output contract. The Owner approved the design and authorized Stage 4 / Phase 6; the Driver recorded the bounded contract in `PHASES.md` and created `tasks/phase-6.md`. Coder implements, independent Watcher verifies, and Owner accepts before any checkpoint or release action.

## Authorization

Owner explicitly authorized Phase 2 on 2026-09-22, with optional environment-sourced bearer authentication, strict TLS, no credential leakage, and an owner acceptance gate before any checkpoint commit or push.

Owner explicitly accepted Phase 2 on 2026-09-23 after successful endpoint testing and final independent Watcher PASS.

Owner explicitly authorized Phase 3 on 2026-10-13: migrate the repository process structure to Factory Core Concepts v1.2 (full conformance; Stage 2 / Phase 3 framing; historical `reports/` preserved local-only). Pre-migration baseline `bf60cc754f5ec0e47233d88839d3c0200993ee3f`, tagged `pre-factory-v1.2`.

Owner explicitly accepted Phase 3 on 2026-10-13 after independent Watcher PASS on AC1–AC8; checkpoint commit and push to `main` authorized with conditions: preserve the verified state; no product code changes; no scope expansion.

Owner explicitly authorized Stage 3 / Phase 4 (`radichat-mock`) on 2026-10-14. Owner accepted Phase 4 on 2026-10-14 after independent Watcher PASS on AC1–AC8; the accepted checkpoint commit and push to the existing `main` branch are authorized. At that point Phase 5 had not yet been authorized; the separate Phase 5 authorization is recorded below.

Owner explicitly authorized Stage 3 / Phase 5 (Adversarial Protocol Investigation) on 2026-10-14, based on accepted Phase 4 checkpoint `8e2951274e52f72963ad8ecb1d75a4a477fe6561`. Phase 5 excludes product-code changes, fixes, and publication. Target Owner acceptance is 2026-10-15 14:00 CEST; showcase materials are for Owner review only.

Owner accepted Stage 3 / Phase 5 on 2026-10-14 after accepting the independent Watcher PASS. The bounded investigation recorded zero confirmed divergences; no product fix was made. The durable handoff and publishable evidence summary are `docs/handoffs/phase-5.md` and `docs/testing-phase-5.md`. The showcase draft is prepared but unpublished; publication requires separate Owner authorization.

Owner approved the v0.1.1 File I/O and Batch design at commit `334e9710efeab981782870df4bc0e050ef195be7` and authorized Stage 4 / Phase 6 on 2026-10-23. The authorization covers implementation and independent Watcher verification; explicit Owner acceptance remains required. Merge, tag, release, and publication are not authorized.

## History

Last completed phase: **Stage 4 / Phase 6 — v0.1.1 File I/O and Batch Support**.

Last accepted phase: **Stage 4 / Phase 6 — v0.1.1 File I/O and Batch Support**.

No phase is currently authorized for execution.

## Authority

The stage/phase roadmap and immutable contracts live in `PHASES.md`.

The product definition and factory rules live in `PROJECT.md`.

Driver behavior lives in `.factory/DRIVER.md` (materialized Factory kit).

Watcher verification behavior lives in `.factory/WATCHER.md` (materialized Factory kit).

Do not create or execute later-phase task files in advance.

Owner explicitly accepted Stage 4 / Phase 6 on 2026-10-23 after reviewing the corrected candidate and fresh independent Watcher PASS. The checkpoint is closed; this acceptance does not authorize tagging, release, image publication, showcase publication, or another phase. See `docs/handoffs/phase-6.md`.
