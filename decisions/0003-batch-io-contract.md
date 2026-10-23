# Batch file I/O and JSONL results

**Status:** Accepted for Stage 4 / Phase 6 implementation. Owner acceptance of the phase is still required before a checkpoint or release.

## Context

RadiChat remains a single-process OpenAI-compatible terminal client. Local batch runs and non-interactive jobs need a system prompt file, a sequential input file, and opt-in complete-result output without a database, a Kubernetes client, or a second conversation runner.

Decision 0001 defines per-request context accounting. Startup must apply that same fit rule to a non-empty system prompt before any request, for both inline and file prompts. This record extends that decision. It does not change it.

## Decision

Relative `system_prompt_file` paths resolve against the lexical directory of the config path. Absolute paths are used as given. `Load` follows symlinks, requires a regular file of at most 1 MiB, requires UTF-8, strips one leading BOM and one trailing LF or CRLF, and rejects an empty result. Errors name the path and the problem and do not include prompt bytes. `system_prompt` and `system_prompt_file` conflict when both keys are present. `Parse` stays pure; only `Load` reads the prompt file.

A non-empty system prompt, inline or from a file, must fit at startup together with an empty user message and `generation_reserve`. No endpoint request is made when it does not. An absent or empty system prompt keeps the previous startup behavior. The system prompt remains after `/clear`.

`-input PATH` is relative to the working directory, including a file named `-` opened as `./-`. It accepts a regular file or a FIFO and rejects a missing path or a directory before any request. The REPL receives an `io.Reader`, not a path.

`-output PATH` is created only after config, prompt fit, and input validation succeed. Creation is exclusive, mode `0600`, and never appends or overwrites. `-output -` writes JSONL only to stdout and keeps diagnostics on stderr. A record-write failure exits 1 and sends no further request. A broken stdout pipe keeps the Go/Unix SIGPIPE default.

JSONL v1 writes one record per handled non-blank physical line. Records are append-only for the process run. Context trimming and `/clear` do not remove earlier records. Successful records contain the complete committed assistant text. Failed or interrupted streams abort the staged turn and do not record partial assistant text as success. SIGINT exits 130 and SIGTERM exits 143; that status wins over an earlier turn failure and over a failed best-effort interruption record.

## Consequences

Batch output can contain prompts and replies, so file output is opt-in and private. It is not a session store and cannot be reloaded. Kubernetes examples are JSON manifests checked structurally; they are not a claim that an image or a cluster was run. Decision 0001 still governs estimation, trimming, and commit. The startup fit check is an additional call to that policy, made once before the REPL starts.
