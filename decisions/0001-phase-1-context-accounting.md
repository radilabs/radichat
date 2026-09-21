# Phase 1 context accounting

**Status:** Accepted

## Context

RadiChat targets arbitrary OpenAI-compatible endpoints with small context windows and minimal dependencies. Exact tokenizer/template accounting is model-specific. The owner explicitly approved the Phase 1 design and its local accounting choice before implementation.

## Decision

Before every request, charge UTF-8 content bytes plus 32 units per message and 32 per request. Reserve `generation_reserve`, and require estimated prompt units plus reserve to fit `context_budget`. Trim oldest complete user/assistant pairs, preserving the system and current user messages. Reject locally if protected messages alone do not fit. Use overflow-safe arithmetic. Commit staged history only after a successful complete stream.

## Consequences

The client deterministically enforces application budget units, not exact model tokens. Documentation must say so. Server context errors are explicit and trigger no hidden fallback/retry. No tokenizer dependency, summarization, or persistence is introduced. Actual server/template usage remains a documented limitation.
