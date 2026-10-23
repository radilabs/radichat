# Stage 4 / Phase 6 — v0.1.1 File I/O and Batch Support

**Status:** Authorized; implementation has not started. Waiting for Owner review of the detailed implementation plan.

**Authorization:** Owner approved design commit `334e9710efeab981782870df4bc0e050ef195be7` and authorized Stage 4 / Phase 6 on 2026-10-23.

**Contract:** `PHASES.md`, “Phase 6 — v0.1.1 File I/O and Batch Support (Stage 4)”.

**Design:** `designs/v0.1.1-file-io-batch-design.md`.

## Work plan

1. Preserve the accepted baseline and identify exact existing config, input, conversation, stream, and doc behavior before editing.
2. Implement `system_prompt_file` in config loading; keep `Parse` pure, read only through `Load`, and test conflicts, config-relative paths, symlinks, regular-file/size/UTF-8/BOM/newline/empty rules, safe errors, and startup context fit for inline and file prompts. No-prompt configurations preserve current behavior.
3. Implement startup validation and `-input PATH`; keep bounded line reading and physical line accounting in a dedicated input component. The REPL receives an `io.Reader`, does not receive file paths, and retains existing stdin and command behavior. Reject missing inputs/directories before requests; allow FIFOs and document their blocking open behavior.
4. Add finish-reason metadata through an additive client API and test that existing client behavior remains compatible.
5. Implement a dedicated JSONL v1 writer and have the REPL emit the contract-defined records. The writer owns JSON encoding and record I/O; `internal/chat` stays free of file I/O. Verify field types/omission, one-write record framing, escaping, output-create-last/exclusive mode 0600, write failure exit 1, stdout-only JSONL, stderr diagnostics, default broken-pipe behavior, and no further requests after record write failure.
6. Verify every handled nonblank line is represented once regardless of conversation context trimming. Specifically process multiple turns that cause prior pairs to be trimmed, then confirm all earlier result records remain complete and ordered in JSONL. Verify failed/interrupted streams abort staged conversation state and never produce successful or partial-output records.
7. Add SIGTERM cancellation that returns exit 143 while preserving SIGINT 130. Test real SIGTERM during an in-flight stream and during idle input; interruption exit status takes precedence over earlier turn failures, and interruption-record failure does not replace it.
8. Add JSON Kubernetes ConfigMap/Job examples and documentation. Validate manifest structure, data/args/mount consistency, prompt path under `/config`, embedded config through the real config parser, and the placeholder image digest. Do not claim a real cluster or image run.
9. Update `PROJECT.md` Capabilities with the approved concise v0.1.1 feature bullet; update CLI/configuration/context-accounting docs and README; add `decisions/0003` recording prompt path/output/schema rules and the startup fit check as an extension of decision 0001.
10. Run and record the required checks, inspect the final diff for scope, and return the exact frozen candidate identity, evidence, limitations, and deferred work to the Driver.

## Required checks

Record actual command and result for each:

- `gofmt` on changed Go files (`gofmt -l .` is empty).
- `go test ./... -count=1`.
- `go test -race ./...`.
- `go vet ./...`.
- `go build ./...`.
- `go run . -help`.
- Kubernetes JSON structural and cross-file/config checks.
- `git diff --check`.
- `sh .factory/check-public-surface.sh`.

## Evidence and decisions log

| Date | Item | Evidence / result |
| --- | --- | --- |
| 2026-10-23 | Phase authorized | Owner-approved design commit `334e9710efeab981782870df4bc0e050ef195be7`; Owner message authorizes implementation and Watcher verification. |
| 2026-10-23 | Architect review | Existing Architect thread reports no remaining design blockers after three bounded review passes. |
| 2026-10-23 | Initial candidate state | Baseline `334e9710efeab981782870df4bc0e050ef195be7`; working tree clean before phase-contract materialization. |
| — | Candidate freeze | Pending implementation completion. |
| — | Watcher | Pending; must record explicit PASS/FAIL and candidate identity. |
| — | Owner acceptance | Pending. |

## Known limits and deferred work

- Real Kubernetes cluster and image execution are not available/authorized; only documentation/manifests and structural checks are in scope.
- Broken stdout pipe behavior follows Go/Unix SIGPIPE defaults; no custom SIGPIPE conversion is in scope.
- Product fixes, session persistence/reload, database, concurrency, retries, scheduling/orchestration, Kubernetes client code, and release/publication are excluded.
- No real image digest exists for v0.1.1; examples must visibly mark the placeholder.
- Additional deferred items: none recorded yet.

## Handoff state

Implementation: not started.

Independent Watcher: not started.

Owner acceptance: pending.

Do not merge, tag, release, or publish without separate Owner authorization. Stop for Owner acceptance after independent Watcher PASS.
