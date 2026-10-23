# CLI behavior

## Invocation

```text
radichat [-config PATH] [-input PATH] [-output PATH|-]
radichat -help
```

Default config path: `radichat.json` in the working directory.

`-input PATH` reads lines from a regular file or FIFO instead of stdin. The path is relative to the working directory. A file named `-` is opened as `./-`. Missing paths and directories exit 1 before any request. File input is non-interactive. Opening a FIFO blocks until a writer connects, and SIGINT or SIGTERM cannot interrupt that open. See [Batch input and JSONL results](batch.md) for the FIFO and signal limitation.

`-output PATH` creates a new mode `0600` JSONL file after config, prompt fit, and input validation. An existing file is left unchanged and startup fails. `-output -` writes JSONL only to stdout. See [Batch input and JSONL results](batch.md).

## Commands

Input is line-oriented. Lines are not sent until newline.

| Input | Behavior |
| --- | --- |
| nonempty text that does not start with `/` | stage and POST a streamed completion |
| blank / whitespace-only | ignored; no request |
| `/clear` | clear in-memory history; keep `system_prompt` |
| `/quit` | exit 0 |
| other `/...` line | error on stderr; not sent to the model |
| EOF | after finishing the current line (if any), exit |
| Ctrl-C | cancel in-flight HTTP, discard the staged turn, exit |

Commands must be the entire trimmed line (`/quit extra` is unknown, not quit). Matching is case-sensitive.

## Streams

* Assistant tokens go to **stdout** with no ANSI wrapping, unless `-output -` is set. In that mode stdout is JSONL only and deltas are not written. In authenticated mode, streamed assistant text and JSONL output are filtered so the bearer token (including a trailing proper prefix) is not printed.
* Prompts (`> `), trim notices, command errors, and diagnostics go to **stderr**.
* Prompts are shown only when stdin and stdout are terminals. `-input` never prompts.
* Redirected stdin is a valid way to drive a short chat; there is no prompt.

## Exit codes

| Code | When |
| --- | --- |
| 0 | `-help`, `/quit`, or EOF after a session with no unrecovered turn failure |
| 1 | invalid/missing config (including an unusable `bearer_token_env` or system prompt file), prompt does not fit, bad `-input` or `-output`, a JSONL record write failed, or redirected/EOF session whose last unrecovered turns failed, or unreadable input |
| 2 | unknown CLI flags or extra arguments |
| 130 | interrupted (Ctrl-C / SIGINT) |
| 143 | SIGTERM |

An interruption exit replaces exit 1 from an earlier failed turn. SIGPIPE on a broken stdout pipe is unchanged.

Startup config errors always exit before the REPL and before any network access. That includes a missing, empty, or malformed token when `bearer_token_env` is set. Diagnostics do not print the token. Per-turn network/protocol errors print to stderr and leave an interactive session running. In redirected/EOF mode, any such failed turn makes the eventual EOF exit code 1.

## Persistence

RadiChat does not write history files, logs, transcripts, or credentials unless `-output` is set. That opt-in JSONL file is a per-run result, not a session that the next process reloads. A second process sharing the same config starts with an empty conversation. Bearer tokens exist only in process memory after startup resolution from the environment and are not written to JSONL, stdout, or stderr.
