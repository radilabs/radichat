# Stage 4 / Phase 6 — v0.1.1 File I/O and Batch Support

**Status:** Corrected candidate has fresh independent Watcher PASS. Owner-authorized review-checkpoint commit and push are in progress; Phase 6 acceptance remains pending Owner inspection.

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
| 2026-10-23 | Plan approved | Owner approved the contract, task file, and seven-task plan at commit `67a573944d9d87313a7aef064e295afda93bbe92`; authorized execution of Tasks 1–7. |
| 2026-10-23 | Coder transport | Existing Coder thread `46c38865-da1d-4849-a2da-65f754043115`, run `run:thread:46c38865-da1d-4849-a2da-65f754043115:ordinal:6`, remained `starting` with `startedAt: null`; bounded wait and recovery did not start a provider turn. `steer` and `restart` returned `thread_not_sendable`. No implementation edits were made. CAO reports no active sessions and has no Phase 6 Coder profile. |
| 2026-10-23 | Coder resumed | Existing Coder thread started run `run:thread:46c38865-da1d-4849-a2da-65f754043115:ordinal:8` through the Cursor provider using `grok-4.7` at 19:40:25 UTC; implementation is now in progress. |
| 2026-10-23 | Candidate freeze | Baseline `67a573944d9d87313a7aef064e295afda93bbe92`. Uncommitted content identity `feab677ce5470af044bcc89bc9905ba82919712076e72c5e8b75796ab66c2ca3` (recipe: SHA-256 of `baseline <HEAD>\n`, raw `git diff HEAD -- . ':!TASKS.md' ':!tasks/phase-6.md'`, then for each NUL-delimited untracked path from `git ls-files --others --exclude-standard -z`, path + LF + lowercase hex SHA-256 of its bytes + LF). Task file and `TASKS.md` are excluded. Driver independently recomputed the digest. No commit. |
| 2026-10-23 | `gofmt -l .` | No paths printed. |
| 2026-10-23 | `go test ./... -count=1 -timeout 300s` | Pass. Packages: root 15.978s, `cmd/radichat-mock` 1.879s, `examples/kubernetes` 0.002s, `internal/chat` 0.004s, `internal/client` 4.283s, `internal/config` 0.017s, `internal/mockllm` 0.019s, `internal/repl` 0.116s. `internal/discovery` has no test files. |
| 2026-10-23 | `go test -race ./... -count=1 -timeout 300s` | Pass. Packages: root 17.002s, `cmd/radichat-mock` 2.893s, `examples/kubernetes` 1.022s, `internal/chat` 1.039s, `internal/client` 5.405s, `internal/config` 1.074s, `internal/mockllm` 1.073s, `internal/repl` 1.160s. |
| 2026-10-23 | `go vet ./...` | Pass. |
| 2026-10-23 | `go build ./...` | Pass. |
| 2026-10-23 | `go run . -help` | Exit 0. Usage lists `-config`, `-input`, `-output`, and `./-`. |
| 2026-10-23 | Kubernetes JSON | `go test ./examples/kubernetes -count=1` pass. Replacing `system_prompt_file` with `/config/missing.txt` failed the test on the prompt path; the manifest was restored and the test passed. No cluster or image run. |
| 2026-10-23 | `git diff --check` | Pass. |
| 2026-10-23 | `sh .factory/check-public-surface.sh` | Pass. No configured Factory runtime paths are tracked. |
| 2026-10-23 | Scope inspection | `go.mod` and `go.sum` unchanged. No new dependency. `internal/chat` has no file I/O. Decision 0001 was not edited. Phase 0–5 task, handoff, and testing evidence files were not modified. |
| 2026-10-23 | Watcher PASS | Existing Watcher thread `c735bfd9-75d3-42a9-9927-82e1dc9a97aa`, run `run:thread:c735bfd9-75d3-42a9-9927-82e1dc9a97aa:ordinal:8`, independently recomputed content identity `feab677ce5470af044bcc89bc9905ba82919712076e72c5e8b75796ab66c2ca3` and returned PASS on AC1–AC11. Report: `.factory/reports/stage-4-phase-6-watcher-01.md` (ignored runtime evidence). Re-run all required gates and independently inspected contract-derived cases. |
| 2026-10-23 | Dr Watson review | No blocking risks; no files changed. Two low residuals assessed as nonblocking and recorded below. |
| 2026-10-23 | Owner correction | Owner authorized a bounded example correction: HTTPS when `bearer_token_env` is set, `automountServiceAccountToken: false` on the Job Pod spec, and no FIFO/SIGTERM implementation change. Product code was not edited. |
| 2026-10-23 | Correction tests | Before the manifest edits, `go test ./examples/kubernetes` failed at `automountServiceAccountToken <nil>`, then, after that field was set, failed on endpoint `http://llm.default.svc.cluster.local/v1/chat/completions`. After the HTTPS edit, `TestKubernetesManifests` passed. No cluster or image run. |
| 2026-10-23 | Corrected candidate | Baseline `67a573944d9d87313a7aef064e295afda93bbe92`. Content identity `7400ede49b71e846a86771ea950eba4da7206a4752c30957191460a5062e3a8b`. Same recipe as the prior freeze. This task file and `TASKS.md` are excluded. No commit. |
| 2026-10-23 | `gofmt -l .` (rerun) | No paths printed. |
| 2026-10-23 | `go test ./... -count=1 -timeout 300s` (rerun) | Pass. Packages: root 15.968s, `cmd/radichat-mock` 1.836s, `examples/kubernetes` 0.004s, `internal/chat` 0.006s, `internal/client` 4.278s, `internal/config` 0.014s, `internal/mockllm` 0.020s, `internal/repl` 0.115s. `internal/discovery` has no test files. |
| 2026-10-23 | `go test -race ./... -count=1 -timeout 300s` (rerun) | Pass. Packages: root 16.991s, `cmd/radichat-mock` 2.869s, `examples/kubernetes` 1.017s, `internal/chat` 1.036s, `internal/client` 5.403s, `internal/config` 1.071s, `internal/mockllm` 1.066s, `internal/repl` 1.153s. |
| 2026-10-23 | `go vet ./...` (rerun) | Pass. |
| 2026-10-23 | `go build ./...` (rerun) | Pass. |
| 2026-10-23 | `go run . -help` (rerun) | Exit 0. Usage lists `-config`, `-input`, `-output`, and `./-`. |
| 2026-10-23 | Kubernetes JSON (rerun) | `go test ./examples/kubernetes -count=1 -v` pass (`TestKubernetesManifests` 0.00s). Assertions require `https://` when `bearer_token_env` is set and `automountServiceAccountToken` false, and still check mount paths and embedded config. No cluster or image run. |
| 2026-10-23 | `git diff --check` (rerun) | Pass. |
| 2026-10-23 | `sh .factory/check-public-surface.sh` (rerun) | Pass. No configured Factory runtime paths are tracked. |
| 2026-10-23 | Correction scope | This correction changed `examples/kubernetes/configmap.json`, `examples/kubernetes/job.json`, `examples/kubernetes/kubernetes_test.go`, and `docs/kubernetes.md`. No RadiChat product source was edited for the correction. |
| 2026-10-23 | Fresh independent Watcher | PASS on corrected identity `7400ede49b71e846a86771ea950eba4da7206a4752c30957191460a5062e3a8b`, baseline `67a573944d9d87313a7aef064e295afda93bbe92`; report `.factory/reports/stage-4-phase-6-watcher-02.md` (ignored runtime artifact). Identity independently recomputed MATCH; all required checks passed. Verification only, not Owner acceptance. |
| — | Owner acceptance | Pending Owner and ChatGPT inspection after review-checkpoint push. |

## Known limits and deferred work

- Real Kubernetes cluster and image execution are not available/authorized; only documentation/manifests and structural checks are in scope.
- Broken stdout pipe behavior follows Go/Unix SIGPIPE defaults; no custom SIGPIPE conversion is in scope.
- Product fixes, session persistence/reload, database, concurrency, retries, scheduling/orchestration, Kubernetes client code, and release/publication are excluded.
- No real image digest exists for v0.1.1; examples must visibly mark the placeholder.
- Non-blocking risk for Owner review: SIGTERM cannot cancel a process blocked opening an input FIFO; it exits 143 only after a writer connects and control reaches the REPL. The Kubernetes example uses a regular input file. No implementation change was made for this limitation.
- The example endpoint is `https://llm.default.svc.cluster.local/v1/chat/completions` because bearer-token auth is configured. TLS certificate provisioning is an operator prerequisite. The bearer-token-over-HTTP observation is resolved for this example. No cluster was contacted.
- The example Job sets `automountServiceAccountToken: false` on `spec.template.spec`. No cluster was contacted to observe that setting.
- The candidate is uncommitted. Identity above excludes this task file and `TASKS.md` so recording the hash does not change it.
- `TestRecordSurvivesTrimAndClear` proves trimming with `trimmed_pairs` and the stderr notice. Under budget 180 the staged request stays at 2 messages, so message-count shrinkage is not the signal.
- `TestJSONLInterruptRecordOmitsPartial` releases the mock handler before server close. Exit 130 was observed and the JSONL record did not contain the partial assistant text.
- SIGPIPE behavior was not given a custom test; it remains the Go/Unix default.

## Handoff state

Implementation: corrected candidate frozen for Driver review. Tasks 1–7 of `docs/superpowers/plans/2026-10-23-v011-file-io-batch.md` remain implemented. The Owner correction changes the Kubernetes example only.

Changed surfaces: config prompt-file loading, startup fit, `-input`, `-output`, REPL line reader and JSONL records, additive `StreamResult`, SIGINT 130 and SIGTERM 143, docs, decision 0003, and `examples/kubernetes`.

Runtime report: `.factory/reports/stage-4-phase-6-coder-evidence.md` (ignored).

Independent Watcher: PASS on the previous identity `feab677ce5470af044bcc89bc9905ba82919712076e72c5e8b75796ab66c2ca3` only; report `.factory/reports/stage-4-phase-6-watcher-01.md` (ignored runtime artifact). That PASS does not cover the corrected candidate.

Corrected content identity: `7400ede49b71e846a86771ea950eba4da7206a4752c30957191460a5062e3a8b`.

Fresh independent Watcher: PASS on corrected identity `7400ede49b71e846a86771ea950eba4da7206a4752c30957191460a5062e3a8b`; report `.factory/reports/stage-4-phase-6-watcher-02.md` (ignored runtime artifact).

Review checkpoint: Owner-authorized commit and push to `main` are in progress. Phase 6 acceptance remains pending Owner inspection.

Do not merge, tag, release, or publish without separate Owner authorization.
