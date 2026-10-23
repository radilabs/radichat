# Batch input and JSONL results

RadiChat can run the same sequential REPL from a file and write one JSON object per handled input line. This is for local batch checks and non-interactive jobs. It does not store a session, retry failed turns, or run requests concurrently.

## Commands

```text
radichat [-config PATH] [-input PATH] [-output PATH|-]
```

With no new flags, stdin and stdout behave as before, except for the startup context-fit check on a non-empty system prompt and SIGTERM exit 143.

`-input PATH` reads conversation lines from a regular file or a FIFO instead of stdin. Paths are relative to the working directory. A file whose name is `-` is passed as `./-`; `-input -` is not an alias for stdin. Opening a FIFO waits until a writer opens it. A missing path or a directory exits 1 and does not contact the endpoint. File input does not print the interactive `> ` prompt.

**FIFO and signal limitation.** The input path is opened before the REPL starts, and that open blocks until a writer connects. SIGINT and SIGTERM are recorded, but they cannot interrupt the blocked open. The process exits with 130 or 143 only after a writer opens the FIFO and the REPL receives the cancellation. Use a regular file when prompt signal exit is required. The Kubernetes example uses a regular file.

Line rules match stdin: one physical line is one turn, blank and whitespace-only lines are ignored, `/clear` and `/quit` work, a final line without a newline is processed, and a line longer than the input bound is rejected. Line numbers count physical lines, including blanks, starting at 1. An unknown `/command` is a failed turn: if it is the last input line, the process exits 1 at EOF.

## Output

`-output PATH` creates a new file with mode `0600`. If the path already exists, startup fails, the file is left unchanged, and no request is sent. RadiChat never appends or overwrites. The file is created only after the config loads, the system prompt fits, and the input path is valid.

`-output -` writes only JSONL to stdout. Streamed assistant deltas are not written there. Diagnostics, the prompt, and trim notices stay on stderr. Every stdout byte in this mode belongs to a JSONL record.

A failure while writing a record exits 1 and does not send another request. If the record is the best-effort interruption record, SIGINT still exits 130 and SIGTERM still exits 143. A broken stdout pipe keeps the usual Go/Unix SIGPIPE behavior.

Terminal streaming remains on stdout when `-output` is a file path. Interactive input can be combined with `-output`.

## JSONL v1

Each handled non-blank line produces one record. Every record has `v` (the integer 1), `line` (the 1-based physical input line number, counting blank lines), and `kind`. Other fields that do not apply are omitted, not null. `line` is present on every kind, including `clear`, `quit`, and `command_error`.

| `kind` | `status` | Fields |
| --- | --- | --- |
| `turn` | `ok` | `input`, `output`, `finish_reason` (`stop`, `length`, or empty), `trimmed_pairs` |
| `turn` | `error` | `sent`, optional `input` (omitted when the line is too long), `error` |
| `turn` | `interrupted` | `sent`, `input` |
| `clear` | absent | none |
| `quit` | absent | none |
| `command_error` | absent | `input` (after CR removal, before trimming), `error` |

`sent` is true only after the HTTP request write succeeds. Local rejection, including a line that does not fit, uses `sent: false`.

Successful `output` is the complete committed assistant text. A failed or interrupted stream aborts the staged turn and does not put partial text in `output`. Records already written stay in the file when later turns are trimmed from model context or when `/clear` runs. `/clear` keeps the system prompt for the next request. `/quit` writes its record and then stops. Lines after `/quit`, after a record-write failure, or after interruption are not recorded. An interrupt while waiting for the next line has no line to record.

JSON encoding does not escape `<`, `>`, or `&` as HTML. Quotes, backslashes, and control characters use normal JSON escapes. UTF-8 is preserved.

The file can contain prompts and replies. It is private because it is opt-in and mode `0600`. It is not reloaded on the next run.

## Signals

| Signal | Exit | Record |
| --- | --- | --- |
| SIGINT | 130 | best-effort `interrupted` record for an in-flight turn |
| SIGTERM | 143 | best-effort `interrupted` record for an in-flight turn |

An interruption exit replaces exit 1 from an earlier failed turn. See [CLI behavior](cli.md).
