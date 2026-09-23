# RadiChat — Stage 1 / Phase 2: Endpoint Authentication

## Authorization and Status

Status: **accepted**.

Owner explicitly authorized Phase 2 on 2026-09-22 and accepted it on 2026-09-23 after successful endpoint testing. Phase 1 accepted baseline: `6468df6bf6ed8dcabd8d7679b73d4494f9c24d2b`.

## Immutable Contract Reference

Authority: `PHASES.md` → Stage 1 / Phase 2. This section restates the contract without amendment.

### Goal

Add optional authentication while preserving the zero-auth local-model path.

### Scope

- Optional bearer-token authentication for OpenAI-compatible endpoints.
- Secret supplied via environment/config indirection that avoids committing credentials.
- Authenticated and unauthenticated modes share the same chat behavior.
- Documentation and tests for both paths.

### Explicit Exclusions

- OAuth/login UI.
- Cloud-provider-specific credential workflows.
- Session persistence.
- Database.
- Compaction/summarization.
- Tools/agents/RAG/MCP.

Owner also explicitly excludes credential storage, TLS skip-verify/insecure modes, custom CA handling, TUI, and later-phase work. Existing HTTPS/TLS behavior stays unchanged.

### Acceptance Criteria

1. RadiChat can call a bearer-token-protected OpenAI-compatible endpoint.
2. Existing unauthenticated local endpoint behavior remains functional.
3. Credentials are not required for local use and are not printed in normal/error output.
4. Authentication behavior is covered by tests.
5. No Phase 2 exclusion is implemented.

### Handoff Contract

Before Phase 2 can be declared complete: all acceptance criteria demonstrated; security-relevant behavior documented; known limitations and deferred work recorded; Watcher verification PASS; owner explicitly accepts. Then STOP.

## Execution Design

Use one optional JSON configuration field naming the environment variable that contains the bearer token (proposed `bearer_token_env`). The config file and public example contain no live token. When the field is absent, retain Phase 1 request headers and zero-auth behavior. When present, validate the variable name, resolve a nonempty token at startup, and send exactly `Authorization: Bearer <token>` on requests to the configured endpoint. Missing/empty values fail before network access. Prevent CR/LF or malformed header values. Never place token values in error strings, logs, test failure messages, docs, reports, or sample configs. Keep the endpoint's existing credential-in-URL rejection and redirect refusal. Do not change the HTTP transport TLS configuration, verification, CA handling, or protocol parser. Keep secret lifetime in process memory only.

The Coder Team may refine the internal representation within this policy. Any compatibility or contract question requiring wider auth behavior returns to Driver; no OAuth, provider-specific flow, token file, credential persistence, or TLS option may be added.

## Authorized Tasks

### 2.0 Driver preflight and transport

- [x] Confirm accepted Phase 1 baseline, clean working tree, Go stack, ignored live config/binaries/runtime reports, and Phase 2 authorization.
- [x] Verify CAO provider profiles and live launches for every delegated CLI role actually used; record bounded recovery/fallback evidence if needed.
- [x] Confirm no actual token value is read or printed during readiness checks.

### 2.1 Coder implementation

- [x] Add optional environment-variable indirection and validation without changing the unauthenticated request path.
- [x] Add bearer header only for the authenticated mode; fail clearly without exposing a token on missing/empty/invalid input or endpoint failures.
- [x] Preserve Phase 1 TLS transport, strict verification, redirect refusal, and no retry behavior.
- [x] Add focused tests for config, protected endpoint, unauthenticated endpoint, no credential leakage, and unchanged chat/stream behavior.
- [x] Update README/configuration/protocol docs and safe public example without credentials.

### 2.2 Driver checks and Dr Watson review

- [x] Inspect actual diff and direct build/test/runtime evidence; run formatting, vet, tests, race checks, build, and black-box auth/no-auth endpoint tests.
- [x] Request Dr Watson review for secret handling, HTTP headers, TLS/redirect boundaries, and omissions. Route in-scope findings for correction and retest.
- [x] Record limitations, deferred work, changed files, commands/results, reviewer dispositions, and transport state.

### 2.3 Independent Watcher and owner gate

- [x] Obtain independent Watcher PASS/FAIL against actual final tree for AC1–AC5, handoff readiness, exclusions, and Stage 1 exit conditions when applicable.
- [x] Re-verify after corrections; no Driver substitution for Watcher.
- [x] Assemble evidence, known limitations/deferred work, review disposition, and Watcher result for owner acceptance; stop at this gate.
- [x] After explicit owner acceptance, create the accepted handoff and checkpoint commit/push authorized by the owner.

## Verification Matrix

| Criterion | Required direct evidence |
| --- | --- |
| AC1 | Built binary completes streamed chat with local bearer-protected test endpoint and expected Authorization header. |
| AC2 | Built binary completes equivalent streamed chat with unauthenticated local endpoint and no Authorization header; Phase 1 tests remain green. |
| AC3 | Missing/empty/invalid token path rejects without network and without displaying token; no token required in local config. |
| AC4 | Focused config/request/stream tests and black-box auth/no-auth cases, including safe failure diagnostics. |
| AC5 | Diff/tree scan confirms all exclusions, strict TLS, no credential storage, and no unrelated work. |

## Progress and Evidence

Authorization recorded and transport readiness complete. Baseline repository was clean at the Phase 1 checkpoint. No actual credential value was read during preflight; tests use harmless sentinels. Coder implementation is complete; ignored `reports/phase-2-coder.md` records its direct evidence. Driver independently ran `gofmt -l .` (empty), `git diff --check`, `go vet ./...`, `go test -timeout 180s -count=1 ./...`, `go test -timeout 180s -count=1 -race ./...`, `go build -o /tmp/radichat-phase2-driver .`, and focused built-binary auth/no-auth/error/redirect/stream-echo/trailing-prefix tests; all passed on the corrected tree. Dr Watson attempt 2 returned PASS. Independent Watcher attempt 1 returned PASS after running the full suite, race checks, build, focused regressions, and a separate seven-case built-binary endpoint harness. The owner tested authenticated operation successfully and accepted Phase 2 on 2026-09-23.

## Changed Files

Lifecycle: `TASKS.md`, `tasks/README.md`, and this task file. Implementation: `internal/config/config.go`, `internal/client/client.go`, `internal/client/stream.go`, `main.go`; tests: `internal/config/auth_test.go`, `internal/client/auth_test.go`, `internal/client/stream_test.go`, `auth_main_test.go`; docs: `README.md`, `docs/configuration.md`, `docs/protocol.md`, `docs/cli.md`. The tracked public example `radichat.example.json` remains unchanged and zero-auth. Transient `reports/phase-2-coder.md` is ignored.

## Known Limitations / Deferred Work

Token source is a process environment variable named by config; users must supply it through their environment. The process environment is not encrypted or hidden from same-user/system inspection. Authenticated stream filtering conservatively replaces any trailing text matching a proper prefix of the token, which may redact harmless text. Verification used local test endpoints; no live third-party model service was used. OAuth/login, token storage/rotation, provider integrations, TLS bypass/custom CA, persistence, databases, compaction, tools/agents/RAG/MCP/TUI, and non-Linux release work remain excluded or deferred.

## Decisions / Regression Memory

Preserve the accepted Phase 1 local request path and strict TLS configuration. Record any durable Phase 2 decision only if later work needs it.

## Watcher / Review Status

Dr Watson attempt 1 returned FAIL in ignored `reports/stage-1-phase-2-watson-01.md`. Driver routed authenticated transport and redirect diagnostics plus streamed-output echo risk to the Coder Team for correction. Those paths now classify errors without untrusted transport/redirect text and filter a complete token or trailing proper token prefix from authenticated stream output, including split deltas. W-001/W-002 were precautionary rather than demonstrated leaks: current startup/config errors contain no resolved token value. W-004 was not a defect: ASCII `~` (0x7E) is printable and valid in an HTTP token. Dr Watson attempt 2 returned **PASS** in ignored `reports/stage-1-phase-2-watson-02.md`; its only note is conservative trailing-prefix over-redaction. Independent Pool Watcher attempt 1 returned **PASS** in ignored `reports/stage-1-phase-2-watcher-01.md` for AC1–AC5 and handoff readiness. The Watcher independently tested the built binary against protected and unprotected endpoints, missing token, redirect with encoded token, split SSE echo, and trailing prefix. Its filename-only credential scan found no common secret patterns in publishable files. Stage 1 goal is demonstrated and Phase 2 is owner-accepted.

## Handoff Status

**ACCEPTED — CHECKPOINT COMMIT/PUSH AUTHORIZED.**

Owner acceptance recorded on 2026-09-23.

## Driver Execution Log

- Preflight: HEAD `6468df6bf6ed8dcabd8d7679b73d4494f9c24d2b`; working tree clean before lifecycle edits. Go 1.27.1 linux/amd64, Cursor CLI, Pool CLI, and Droid CLI installed. Existing live config, binary, Pool runtime state, and reports are ignored; no credential values inspected.
- CAO main service and isolated Pool service both reachable after sandbox-safe loopback escalation. Registered Phase 2-specific profiles: `radichat_phase2_coder` (`cursor_cli`, `cursor-grok-4.6-high`), `radichat_phase2_reviewer` (`droid_cli`, `custom:Step-3.7-Flash-0`), and `radichat_phase2_watcher` (`pool_cli`, configured Poolside model). Profiles restrict role/phase authority and forbid commits/pushes.
- Read-only transport smoke: coder session `cao-radichat-p2-coder` / terminal `920271c1` returned explicit READY, confirmed role, baseline, task file, and no implementation. Reviewer first smoke `cao-radichat-p2-reviewer-smoke` / `19661758` reached the model but asked for exact role wording; bounded fresh retry `cao-radichat-p2-reviewer-smoke2` / `9b954894` returned explicit READY as Reviewer / Dr Watson after the Driver specified `tasks/phase-2.md`. Pool Watcher `cao-radichat-p2-watcher` / `6a795025` read its contract/task file and verified read-only repository/toolchain access. No role inherited another role, and no product change occurred in smoke checks.
- Implementation assigned to the existing Cursor Coder terminal `920271c1` after readiness, with no Factory/lifecycle edit, commit, push, or later-phase authority. Driver additionally flagged that authenticated error responses must not echo a bearer token through Phase 1's bounded server-body diagnostic path; use harmless sentinel tests and preserve no-auth diagnostics.
- Dr Watson attempt 1 through CAO reviewer terminal `1fb514e1` returned FAIL. Driver reviewed each finding, rejected unsupported/incorrect token-parser and ASCII-tilde claims, and routed concrete authenticated transport/redirect diagnostic plus normal streamed-output echo risks to the Coder Team through its existing CAO terminal. Review FAIL remains recorded until a fresh review passes; Watcher has not run on the corrected tree.
- Coder corrected authenticated transport and redirect diagnostics, complete-token SSE echoes across deltas, and trailing proper token-prefix output. Driver independently reran `gofmt -l .`, `git diff --check`, `go vet ./...`, full `go test`, full `go test -race`, `go build`, and focused built-binary auth/no-auth/echo/redirect/transport tests on this corrected tree; all passed. Fresh Dr Watson review terminal `e64b1d60` is in progress. No commit or push.
- Dr Watson attempt 2 through CAO reviewer terminal `e64b1d60` returned explicit PASS. Independent Pool Watcher terminal `6a795025` returned explicit PASS with the required ignored report. Watcher ran its own seven-case built-binary endpoint harness in `/tmp`, full/race tests, and filename-only credential scan. A proposed scan that would have printed matched credential lines was rejected and replaced with the safe filename-only scan. No product edits occurred after the Watcher PASS; only this lifecycle evidence and task-index status were updated. No commit, push, accepted handoff, or next-phase task was created.
- Owner confirmed successful authenticated endpoint operation, accepted Phase 2, and authorized commit/push to `main` on 2026-09-23. `docs/handoffs/phase-2.md` records the accepted Stage 1 checkpoint. No later phase is authorized.
