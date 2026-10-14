# RadiChat — Stage 3 / Phase 5: Adversarial Protocol Investigation

## Authorization and Status

Status: **accepted** (Owner acceptance: 2026-10-14; independent Watcher: PASS).

Accepted prerequisite: Stage 3 / Phase 4 at `8e2951274e52f72963ad8ecb1d75a4a477fe6561`. The immutable Phase 5 contract is in `PHASES.md`. This task decomposes that contract and does not authorize product changes, fixes, or publication.

**Target:** complete investigation and Owner acceptance by **2026-10-15 14:00 CEST**. Prepare draft showcase materials by 15:00 CEST for Owner review at 16:00 CEST. These are targets; publication remains exclusively Owner-authorized.

## Role and Transport Plan

- **Architect:** existing T3 Architect thread; define test matrix and protocol oracle from `docs/protocol.md`.
- **Worker:** Step 5 Preview via the existing Hermes CLI session `20261013_095745_21eecc` (“Step 5 Build Week”), working only on the authorized test harness and durable findings. No new T3 role thread.
- **Luna 6:** separate-context T3 delegated test role, independent oracle and reproduction; no repository edits.
- **Dr Watson:** existing T3 DrWatson thread reviews candidate scope and evidence.
- **Watcher:** existing T3 Watcher thread independently verifies the frozen candidate and findings.

Transport is evidence only; Driver inspects repository state, commands, and role outputs. Session status does not establish task completion or PASS.

## Baseline and Boundaries

- Baseline: accepted Phase 4 commit `8e2951274e52f72963ad8ecb1d75a4a477fe6561` on `main`.
- Oracle: `docs/protocol.md`, with the specific byte-accounting and CLI details in `docs/context-accounting.md` and `docs/cli.md`.
- Test-only harness: `internal/discovery/` with `//go:build mockdiscovery`; use a temporary built RadiChat binary and loopback `radichat-mock` server.
- New scenario payloads required for discovery are registered only by test code and prefixed `p5-`. Do not edit accepted `internal/mockllm/` or `cmd/radichat-mock/`.
- Sentinel credentials only: `mockllm.SentinelToken` and `mockllm.SentinelMismatchToken`. Never write token values into reports/findings.
- Do not modify product code, protocol docs, accepted Phase 0–4 records/evidence, release/container definitions, or showcase/public surfaces. Do not fix defects.
- F-0001 and F-0002 are unrelated, preserved, and out of scope.

## Must-run Matrix

Use exact IDs and record every case outcome. The protocol documentation is the binding oracle; error wording is advisory unless the docs promise that wording.

| IDs | Cases | Expected outcome |
|---|---|---|
| S1–S8 | `stream-success-basic`, `-multibyte`, `-role-only`, `-crlf`, `-comments`, `-no-content-type`, `-empty-content`, `-length-finish` | Successful completion; exact assistant output where applicable. A `length` finish commits streamed text. |
| F1–F6 | `fragment-utf8-inside-rune`, `-inside-crlf`, `-inside-data-field`, `-inside-done`, `-byte-by-byte`, `-multiple-events-one-write` | Same success/output as the unfragmented stream. Do not assert one TCP read per mock write. |
| M1–M11 | `malformed-json-event`, `malformed-delta-content-not-string`, `malformed-multiple-choices`, `malformed-early-termination`, `malformed-early-termination-mid-event`, `stream-error-object`, `stream-error-non-object`, `stream-tool-calls-rejected`, `stream-function-call-rejected`, `stream-finish-content-filter`, `stream-finish-unsupported` | Turn fails; partial output may already be visible; failed turn is not committed. |
| M12 | `stream-error-null-ignored` | Success; `error: null` is not an error. |
| H1–H6 | `http-error-401`, `-429`, `-500`, `-404-unknown-model`, `non-sse-content-type`, `non-sse-empty-body` | Turn fails; no retry. In zero-auth mode non-success diagnostics include only a bounded snippet. |
| R1–R3 | `redirect-to-final-url`, `redirect-with-query`, `redirect-absolute` | Refused; exactly one request per attempted turn. In zero-auth mode Location diagnostics follow the documented redaction behavior. |
| A1–A3 | `auth-required-match`, `auth-required-missing`, `auth-required-mismatch` | Match succeeds; missing/mismatch fail. No Authorization header in zero-auth mode. Authenticated errors omit untrusted bodies. |
| A4–A7 | Startup config with unset token variable, empty value, malformed value, or invalid variable name | Startup fails before network access; zero records; diagnostics contain no token value. |
| E1–E7 | Harness-local token echo in a delta; token split across deltas; trailing token prefix; 401/500 body; stream error object; redirect Location; E1 in zero-auth mode | Authenticated paths never print the sentinel and preserve ordinary surrounding text as documented. Zero-auth stream text is unchanged. Redirect is not followed. |
| C1–C2 | `context-max-tokens-sent`, `context-no-optional-fields` | Correct `max_tokens`, `stream`, model, headers, and absence of `Cookie`, `Authorization` (zero-auth), tools/functions/tool_choice/n. |
| C3–C4 | C3: `context-exact-budget-boundary`; C4: `context-utf8-byte-accounting`. Each case runs an exact-fit input and a one-byte-over input (two evidence rows per case ID). | Exact fit sends; one byte over fails locally with zero records. Charges UTF-8 bytes per documented formula. |
| C7–C8 | `context-pair-trimming` at budget 303 and 302 with system prompt and 20-byte turns | 303 retains complete history; 302 drops the oldest complete pair and reports trimming. Never send a partial pair. |
| C9 | `context-failed-turn-rollback`, two piped input lines | Second request contains the prior committed context plus new user message, but not the failed turn's assistant text. |
| C10 | `context-after-clear`, input `one\n/clear\ntwo\n` | History resets and system message remains. |
| C11 | Harness-local `p5-output-cap`, response exactly `context_budget` bytes and one byte over | Exact cap succeeds; over-cap fails and is not committed. |

For black-box cases use the same harness shape: build `radichat` to a temporary path once; run piped stdin with per-case temporary config; capture exit, stdout, stderr, and sanitized request records; 15-second timeout per case except explicitly marked optional slow cases. Do not persist raw request bodies or secrets.

## Optional / Exploratory Cut

Drop these before cutting Luna or Watcher work. Optional documented edge cases, in order: event larger than 1 MiB; stream payload larger than 8 MiB; line longer than `max(context_budget, 4096)`; system prompt alone over budget; HTTP 204 with empty body; 100 KiB non-success body snippet cap; slow drip and idle stall; extra redirect codes; byte-fragmented auth echoes.

Undocumented probes (UTF-8 BOM, CR-only framing, trailing bytes after `[DONE]`, `data:` spacing/multiple fields, Content-Type parameters/case, invalid UTF-8 JSON, `[DONE]` without prior choices) are exploratory observations only. Do not file them as defects without a documented oracle and separate Owner authorization.

## Outcome and Evidence Rules

- Use verdicts `PASS`, `DIVERGENCE`, `EXPLORATORY`, or `NOT RUN` per case.
- A failed Phase 5-tagged test may be expected when it reproduces a documented divergence. Record it; link it to a finding. The default `go test ./...` must remain green.
- Record sanitized JSONL at `.factory/reports/stage-3-phase-5-run-NN.jsonl`: case/scenario, config hash, stdin, environment variable names (never values), exit code, stdout hash plus bounded preview, stderr bounded preview, sanitized request path/roles/field names/auth outcome, oracle citation, verdict, commit, and Go version. Do not include raw request bodies.
- Each verified documented divergence gets one `findings/F-000N-<slug>.md`, status `new`, with exact command, case ID/scenario, environment/config (no secret values), expected behavior with protocol citation, actual behavior, evidence path, impact/severity, and sentinel scan result. Use the next available ID starting at F-0003. No finding for a hypothesis or doc-silent behavior.
- No minimum number of findings. No false-positive findings to satisfy a showcase narrative.

## Verification and Handoff Steps

1. Freeze baseline; record Go version and commit. Complete the test-only harness and must-run cases through the Hermes Worker.
2. Inspect every finding and evidence artifact in the Driver role; verify reproduction locally without changing product code.
3. Have Luna 6 first derive an independent oracle and select at least three protocol-documented additional cases not selected by Step 5. Then, after the Worker result exists, use a separate delegated test round to reproduce each finding from a clean state and rerun selected cases with a raw `net.Listener` harness independent of `mockllm`.
4. Have Dr Watson review the frozen candidate and any concerns; correct only test/evidence defects in scope. Freeze a new identity after corrections.
5. Have the existing Watcher derive its own oracle from docs before reading the Worker’s expected fields; verify all acceptance criteria, findings, selected extra cases, source boundary, and sentinel scan. Record explicit PASS/FAIL.
6. Run and record: `go test -tags mockdiscovery ./internal/discovery -count=1 -run 'P5' -v` (document divergence-induced nonzero exit), `go test ./... -count=1`, `go vet ./...`, `go build ./...`, `gofmt -l .`, `git diff --check`, and `sh .factory/check-public-surface.sh`.
7. Prepare `.factory/reports/stage-3-phase-5-discord-draft.md` as an unpublished concise showcase draft. Include implementation from accepted Phase 4, reproducible confirmed findings or a zero-finding outcome, evidence/test results, and a short accurate description of the multi-agent workflow. Do not send or publish it.
8. Present task status, findings, Worker/Luna/Dr Watson/Watcher evidence, limits, and draft to the Owner by the target time. STOP at Owner acceptance. No commit/push/phase transition/publication unless separately authorized.

## Current Evidence State

- Architect: matrix and protocol-oracle proposal consulted and adopted; thread `3fd9e604-9de3-48c6-a163-a11da523a5a0`.
- Worker: complete in existing Hermes CLI session `20261013_095745_21eecc`; reports are `.factory/reports/stage-3-phase-5-worker.md` and `.factory/reports/stage-3-phase-5-worker-correction-01.md`. Corrected authoritative run: `.factory/reports/stage-3-phase-5-run-06.jsonl`.
- Driver verification: run-05 independently reran the tagged suite; default tests, vet, tagged vet, build, formatting, diff check, and public-surface check passed.
- Luna 6: independent oracle plus five extra documented cases passed; no Phase 5 finding reproduced. Report: `.factory/reports/stage-3-phase-5-luna-01.md`.
- Dr Watson: both review rounds are complete; reports are `.factory/reports/stage-3-phase-5-watson-01.md` and `.factory/reports/stage-3-phase-5-watson-02.md`. The corrected candidate has no remaining required corrections.
- Candidate identity: final Watcher candidate is frozen; its digest and exact file list are recorded in `.factory/reports/stage-3-phase-5-candidate-freeze.md`.
- Findings: none for Phase 5; existing F-0001 and F-0002 remain preserved and unrelated.
- Watcher: PASS on all eight acceptance criteria; report `.factory/reports/stage-3-phase-5-watcher-01.md` is ephemeral. The tracked, publishable test summary is `docs/testing-phase-5.md` and does not require runtime reports.

## Owner Acceptance and Handoff

- Owner accepted Phase 5 on 2026-10-14 after accepting Watcher PASS.
- After acceptance, Owner separately authorized the Driver to record the final handoff, commit and push the accepted state to `main`, and prepare the showcase package for review.
- Outcome: zero confirmed divergences; no Phase 5 finding or product fix. The three initial candidates were withdrawn as harness/fixture errors.
- Accepted Phase 0–4 evidence and product code remain unchanged.
- Bounded omissions and exploratory-only observations are summarized in `docs/testing-phase-5.md`; per-case runtime records remain disposable under `.factory/reports/`.
- Durable handoff: `docs/handoffs/phase-5.md`.
- Showcase materials: `docs/showcases/phase-5-discord-draft.md`; prepared and unpublished. Publication requires separate Owner authorization.
- No Phase 6 or product fixes are authorized. No phase is currently authorized.
