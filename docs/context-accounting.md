# Context accounting

**Decision:** `decisions/0001-phase-1-context-accounting.md` (accepted).

RadiChat enforces an application working budget. It does **not** count exact model tokens and does **not** reproduce a server's chat template. A server context overflow remains a visible error with no retry or hidden fallback.

## Units

Before each request, estimated prompt units plus `generation_reserve` must be `<= context_budget`.

For a candidate message list:

* charge **UTF-8 content bytes** of each message (`len` of the Go string / the raw text, not JSON-escaped size)
* add **32 units per message**
* add **32 units per request**

Arithmetic is overflow-safe. Overflow is an error.

## Trimming

Conversation state is an optional fixed system message plus zero or more complete user/assistant pairs, held only in process memory.

On each user line:

1. Stage the new user message on a copy of history.
2. If the system message plus that user message (plus reserve) cannot fit, fail locally. No HTTP request is sent and committed history is unchanged.
3. Otherwise drop the oldest complete user/assistant pairs until the request fits.
4. Print a brief notice on stderr when pairs are dropped.
5. Send the staged messages.

The system message and the current user message are never trimmed.

## Commit and rollback

The trimmed history, new user message, and assistant reply are committed only after a stream completes successfully.

Partial assistant text may already have been printed. Cancellation, protocol errors, idle timeouts, and local output-bound errors abort the staged turn and restore the previous committed conversation.

`/clear` drops committed and staged turns and keeps the configured system prompt.

A new process always starts empty. RadiChat does not write history, transcripts, or session files.

## What this does not guarantee

`generation_reserve` is also sent as `max_tokens`. That is a protocol field, not a promise that server-side tokens equal RadiChat units. Different models tokenize and wrap templates differently. Size the budget conservatively for small local windows, and treat a server context error as a real failure.
