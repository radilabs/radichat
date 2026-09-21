# CLI behavior

## Invocation

```text
radichat [-config PATH]
radichat -help
```

Default config path: `radichat.json` in the working directory.

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

* Assistant tokens go to **stdout** with no ANSI wrapping.
* Prompts (`> `), trim notices, command errors, and diagnostics go to **stderr**.
* Prompts are shown only when stdin and stdout are terminals.
* Redirected stdin is a valid way to drive a short chat; there is no prompt.

## Exit codes

| Code | When |
| --- | --- |
| 0 | `-help`, `/quit`, or EOF after a session with no unrecovered turn failure |
| 1 | invalid/missing config, or redirected/EOF session whose last unrecovered turns failed, or unreadable input |
| 2 | unknown CLI flags or extra arguments |
| 130 | interrupted (Ctrl-C / SIGINT) |

Startup config errors always exit before the REPL. Per-turn network/protocol errors print to stderr and leave an interactive session running. In redirected/EOF mode, any such failed turn makes the eventual EOF exit code 1.

## Persistence

RadiChat does not write history files, logs, or transcripts. A second process sharing the same config starts with an empty conversation.
