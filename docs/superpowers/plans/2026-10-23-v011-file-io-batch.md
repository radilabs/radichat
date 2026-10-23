# RadiChat v0.1.1 File I/O and Batch Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task. Each task remains bounded by the active Factory contract in `PHASES.md` and `tasks/phase-6.md`.

**Goal:** Add config-backed prompt files, file-driven sequential REPL input, opt-in complete-result JSONL output, and documentation-only Kubernetes Job examples.

**Architecture:** `main` validates startup inputs and opens files, then passes an `io.Reader` and a JSONL record writer to the existing REPL. A dedicated bounded line reader owns line numbers and limits; a dedicated JSONL writer owns encoding and writes. Conversation staging/trim/commit/abort stays in the existing REPL and `internal/chat`, and result records are never coupled to model context retention.

**Tech Stack:** Existing Go toolchain and standard library only; existing `radichat-mock` for deterministic streamed tests; JSON manifests checked with `encoding/json`.

**Spec:** `designs/v0.1.1-file-io-batch-design.md`; active boundary: Stage 4 / Phase 6 in `PHASES.md` and `tasks/phase-6.md`.

## Global Constraints

- Preserve the existing OpenAI-compatible inference protocol and default stdin behavior, except the approved startup fit check for non-empty system prompts and SIGTERM exit behavior.
- No new dependency, database, transcript persistence/reload, concurrency, retry, scheduler, orchestration, Kubernetes API integration, image build, merge, tag, or release.
- `internal/chat` does no file I/O; `config.Parse` stays pure.
- `-output PATH` is exclusive-create mode `0600`; validate config/prompt/fit/input before creating it.
- `-output -` writes JSONL only to stdout; diagnostics remain on stderr.
- Never write partial or failed assistant streams as successful results; always preserve records across context trimming.
- SIGINT exits 130; SIGTERM exits 143.
- Owner acceptance and independent Watcher PASS are required before phase completion; no checkpoint commit/tag/release before acceptance.

## Review Focus

- **Prompt-fit boundary:** exact fit and one-unit overfit for both inline and file prompts, while no-prompt configurations retain behavior; test with zero requests.
- **Output creation order:** late startup failures must not leave an output file; test missing input together with `-output PATH`.
- **Stream rollback:** partial content followed by stream error or cancellation never becomes JSONL output and never commits conversation state.
- **Trimming independence:** turn output history is append-only for a run even when older message pairs are removed from model context; test turns before and after trimming and `/clear`.
- **Signal/output edges:** SIGINT and SIGTERM produce their specified exit codes, and JSONL writer failures prevent another request without replacing the signal exit code when interruption is active.

---

### Task 1: Config-backed system prompt files and startup budget validation

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`
- Modify: `main.go`
- Test: `main_test.go` (create if needed)
- Reference: `internal/chat/conversation.go`, `decisions/0001-phase-1-context-accounting.md`

**Interfaces:**
- Add `SystemPromptFile string` to `config.Config` as load metadata if useful; the final runtime `Config.SystemPrompt` contains the loaded prompt text.
- Keep `config.Parse(path string, data []byte) (Config, error)` pure; `config.Load(path string) (Config, error)` resolves and reads prompt file.
- Startup fit uses `chat.Fits(messages []chat.Message, budget, reserve int64)` before REPL start.

- [ ] **Step 1: Add failing config tests** for file-only configuration, inline/file mutual exclusion including inline empty, null/non-string/empty file paths, relative path from config directory, absolute paths, symlink following, directory/FIFO rejection, 1 MiB limit, invalid UTF-8, one BOM/terminal newline stripping, empty result, and errors that do not include prompt bytes.
- [ ] **Step 2: Run** `go test ./internal/config -run 'SystemPromptFile' -count=1`; confirm the new cases fail for missing behavior.
- [ ] **Step 3: Implement prompt-file parsing/loading** in `internal/config/config.go`. `Parse` validates/stores the path and conflict; `Load` resolves relative paths against lexical `filepath.Dir(configPath)`, opens/follows symlinks, checks regular file and size using existing limits, validates UTF-8, strips exactly the designed BOM/newline, and returns safe path-specific errors.
- [ ] **Step 4: Add failing startup-fit tests** for empty/no prompt, inline prompt exact-fit/overfit, and file prompt exact-fit/overfit; assert an overfit starts no mock request.
- [ ] **Step 5: Implement the startup fit check** in startup orchestration using existing `chat.Fits`; skip when there is no non-empty system prompt; exit 1 before opening the JSONL output or issuing a request.
- [ ] **Step 6: Run** `go test ./internal/config ./... -run 'SystemPrompt|PromptFit' -count=1`; confirm focused tests pass.

### Task 2: Bounded file input component and startup options

**Files:**
- Create: `internal/repl/input.go`
- Modify: `internal/repl/repl.go`
- Test: `internal/repl/input_test.go` (create)
- Modify: `internal/repl/repl_test.go`
- Modify: `main.go`
- Test: `main_test.go`

**Interfaces:**
- Add `inputReader` in `internal/repl/input.go` wrapping `*bufio.Reader`, maximum bytes, and physical line number; `ReadLine(ctx context.Context) (inputLine, error)` returns line number, line text after current CR removal, and the existing EOF/too-long semantics.
- Keep the REPL path-based-free: startup opens `-input PATH`, passes its `io.Reader` to `repl.RunWithOptions` (or equivalent), and retains `repl.Run` as a compatibility wrapper for existing tests/callers.
- CLI options: `-input PATH`; path flags are relative to CWD; reject missing input/directories before output creation; allow regular files and FIFOs.

- [ ] **Step 1: Add failing bounded-reader tests** for sequential lines, physical numbering including blank lines, EOF with and without final LF, CRLF/CR handling identical to current REPL, exact limit, overlong line drain, cancellation, and FIFO-compatible reader semantics.
- [ ] **Step 2: Run** `go test ./internal/repl -run 'InputReader' -count=1`; confirm failure.
- [ ] **Step 3: Extract bounded line reading** into `inputReader` without changing the existing stdin line behavior; route REPL reads through it.
- [ ] **Step 4: Add CLI tests** for `-input` file use, noninteractive mode, missing path and directory rejection with zero requests, and `-output PATH` not created when input validation fails. Add/help-check `-help` text and `./-` handling.
- [ ] **Step 5: Add `-input PATH` startup handling** in `main.go`; open input before JSONL output, keep stdin default, and pass the `io.Reader` (not its path) to the REPL.
- [ ] **Step 6: Run** `go test ./internal/repl ./... -run 'Input|CLI|Help' -count=1`; confirm focused tests pass.

### Task 3: Additive stream result metadata

**Files:**
- Modify: `internal/client/client.go`
- Modify: `internal/client/stream.go`
- Test: `internal/client/client_test.go`
- Test: `internal/client/stream_test.go`

**Interfaces:**
- Add `StreamResult` with `Text string`, `FinishReason string`, and `RequestSent bool`; `RequestSent` is true only when the client observes a successful `httptrace.WroteRequest` callback (no error).
- Add `(*Client).StreamResult(ctx context.Context, messages []chat.Message, outputLimit int64, onDelta func(string) error) (StreamResult, error)` (or an equivalently named additive method); preserve existing `Stream(...) (string, error)` as a wrapper returning `result.Text`.
- Preserve redaction of streamed and assembled text and existing error behavior.

- [ ] **Step 1: Add failing tests** that read finish reason `stop`, `length`, and empty; assert `RequestSent` is false before successful HTTP request write and true after it; ensure unsupported finish reasons remain errors; and verify the old `Stream` signature/behavior still works.
- [ ] **Step 2: Run** `go test ./internal/client -run 'FinishReason|StreamCompatibility' -count=1`; confirm failure.
- [ ] **Step 3: Return finish reason additively** from stream decoding and expose it through the new result method; preserve the existing method as a wrapper.
- [ ] **Step 4: Run** `go test ./internal/client -count=1`; confirm client tests pass.

### Task 4: JSONL v1 writer and REPL records

**Files:**
- Create: `internal/repl/jsonl.go`
- Test: `internal/repl/jsonl_test.go` (create)
- Modify: `internal/repl/repl.go`
- Test: `internal/repl/repl_test.go`
- Modify: `main.go`
- Test: `main_test.go`

**Interfaces:**
- Define `Record` with JSON fields from the Phase 6 table: `v`, `line`, `kind`, optional `status`, `sent`, `input`, `output`, `finish_reason`, `trimmed_pairs`, `error`; use `omitempty` and pointer/optional fields where needed to omit absent values.
- Define `RecordWriter` with `WriteRecord(Record) error`; `JSONLWriter` marshals with `SetEscapeHTML(false)`, appends one LF, then performs exactly one `Write` call.
- Add a REPL options entrypoint accepting the writer and JSONL-only stdout flag; keep existing `Run` wrapper behavior unchanged when no new options are selected.

- [ ] **Step 1: Add failing writer tests** for each record shape, absent-vs-null fields, JSON escaping/round-trip (quotes, backslash, controls, `</>&`, multibyte UTF-8), single-write framing, short write, and writer error.
- [ ] **Step 2: Run** `go test ./internal/repl -run 'JSONLWriter' -count=1`; confirm failure.
- [ ] **Step 3: Implement `Record` and `JSONLWriter`** in `internal/repl/jsonl.go` with schema v1 and one write per record.
- [ ] **Step 4: Add failing REPL integration tests** for success, local preflight failure (`sent:false`), HTTP/stream failure (`sent:true` when a request starts), command error, `/clear`, `/quit`, blank lines, line numbering, EOF without newline, and one record for every handled nonblank line.
- [ ] **Step 5: Emit records from the existing REPL coordinator** only after the corresponding outcome is known. Success is emitted only after `conv.Commit`; any stream failure aborts and emits error with no output. Output rows are accumulated independently of the trimmed conversation message list.
- [ ] **Step 6: Add tests** proving records for early turns remain after those turns are trimmed from model context and that `/clear` does not erase prior run records while the next request preserves the system prompt.
- [ ] **Step 7: Add `-output PATH|-` handling** in `main.go`. Validate config/prompt/fit/input first; create path output last with `O_CREATE|O_EXCL` and mode `0600`. For `-output -`, route JSONL to stdout, suppress stream deltas and their newline, and keep diagnostics on stderr.
- [ ] **Step 8: Add output safety tests**: existing output is untouched and causes zero requests; startup failure does not leave a file; piping `-output -` to `/dev/full` yields exit 1 and no subsequent request; `-output -` stdout contains only valid JSONL; output file uses mode 0600.
- [ ] **Step 9: Run** `go test ./internal/repl ./... -run 'JSONL|Record|BatchOutput' -count=1`; confirm focused tests pass.

### Task 5: Interrupt handling, rollback, and secret safety

**Files:**
- Modify: `main.go`
- Modify: `internal/repl/repl.go`
- Test: `main_test.go`
- Test: `internal/repl/repl_test.go`
- Test: `internal/client/auth_test.go`

**Interfaces:**
- Use signal-aware cancellation that distinguishes `os.Interrupt` from `syscall.SIGTERM`; REPL returns 130 for SIGINT and 143 for SIGTERM.
- A stream interrupted while a turn is staged calls `conv.Abort`; any JSONL interruption record is best effort and never includes partial output.

- [ ] **Step 1: Add failing subprocess tests** sending real SIGINT and SIGTERM during an active mock stream; assert cancellation, no successful/partial output record, no committed staged turn, and exit codes 130/143.
- [ ] **Step 2: Add redaction integration tests** with a sentinel bearer token echoed into assistant text/errors; assert it is absent from JSONL output/error, stdout, and stderr.
- [ ] **Step 3: Implement signal-aware context handling** for SIGINT and SIGTERM. Ensure an interruption takes precedence over previous turn failures and over failure to write the best-effort interruption record.
- [ ] **Step 4: Run** `go test ./... -run 'Signal|Interrupt|Redact|Rollback' -count=1`; confirm focused tests pass.

### Task 6: Kubernetes JSON examples and durable docs

**Files:**
- Create: `examples/kubernetes/configmap.json`
- Create: `examples/kubernetes/job.json`
- Create: `docs/batch.md`
- Create: `docs/kubernetes.md`
- Modify: `docs/cli.md`
- Modify: `docs/configuration.md`
- Modify: `docs/context-accounting.md`
- Modify: `README.md`
- Modify: `PROJECT.md`
- Create: `decisions/0003-batch-io-contract.md`
- Test: `examples/kubernetes/kubernetes_test.go` (or equivalent focused Go test)

**Interfaces:**
- ConfigMap keys are `radichat.json`, `system-prompt.txt`, and `input.txt`; the Job args use `/config/radichat.json`, `/config/input.txt`, and `-output -`; `system_prompt_file` resolves to `/config/system-prompt.txt`.
- The image reference contains `REPLACE_WITH_V0_1_1_DIGEST` visibly in its digest field; no real image release is claimed.
- `decisions/0003` records prompt path resolution, file/output safety, JSONL v1, startup validation, and that the startup fit check extends decision 0001 without editing it.

- [ ] **Step 1: Add a failing standard-library manifest test** that parses both JSON files, validates required Job/Pod security and retry fields, cross-checks args/data/mount keys and `/config` paths, parses embedded config with `config.Parse`, and asserts the placeholder image marker.
- [ ] **Step 2: Run** `go test ./examples/kubernetes -count=1`; confirm missing-manifest failure.
- [ ] **Step 3: Add the JSON ConfigMap and Job examples** with read-only config mount, nonroot UID 65532, read-only root filesystem, no privilege escalation, all capabilities dropped, no stdin/TTY, restart `Never`, backoff 0, active deadline, stdout result collection, and optional Secret-backed token reference.
- [ ] **Step 4: Add the cross-file manifest test** and confirm it catches a deliberately mismatched path/data key locally before restoring the valid manifest.
- [ ] **Step 5: Document** file prompts, path bases, limits, startup fit rules, batch input semantics, JSONL schema/privacy/error behavior, signal codes, output-file safety, stdout/stderr routing, and Kubernetes log stream caveats.
- [ ] **Step 6: Add the approved capability bullet** to `PROJECT.md`; create decision 0003 referencing decision 0001, without changing decision 0001.
- [ ] **Step 7: Run** `go test ./examples/kubernetes -count=1`; confirm structural checks pass.

### Task 7: Full verification and handoff evidence

**Files:**
- Modify: `tasks/phase-6.md`
- Runtime report: `.factory/reports/stage-4-phase-6-coder-evidence.md`

- [ ] **Step 1: Run** `gofmt -l .` and format only changed Go files until it prints no paths.
- [ ] **Step 2: Run** `go test ./... -count=1`, `go test -race ./...`, `go vet ./...`, and `go build ./...`; record exact outcomes.
- [ ] **Step 3: Run** `go run . -help`, `git diff --check`, and `sh .factory/check-public-surface.sh`; record exact outcomes.
- [ ] **Step 4: Inspect** the complete diff for forbidden dependencies, products outside Phase 6, changes to accepted Phase 0–5 evidence, `.factory/` tracking, and release/container inclusion of development-only examples/tools.
- [ ] **Step 5: Record** baseline, final candidate tree/commit identity as available, changed files, tests, test evidence, known limits, and deferred work in `tasks/phase-6.md`; return the candidate to Driver for review, Dr Watson, and independent Watcher verification.

## Plan self-review

- Contract coverage: prompt loading/fit is Task 1; file input and startup ordering Task 2; finish metadata Task 3; records, trimming independence, error writes and stdout routing Task 4; signal/rollback/redaction Task 5; Kubernetes and durable docs Task 6; all required gates and evidence Task 7.
- The two Owner-specific result invariants have direct tests in Task 4: complete run records survive context trimming, and failed/interrupted streams cannot emit output as success.
- Design boundaries are retained: only one Coder Team writes; no parallel work or integration manifest is needed; Watcher verifies a frozen candidate independently after Driver review.
- Kubernetes verification is structural only; no cluster/image run is claimed.
