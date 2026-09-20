# RadiChat

RadiChat is a deliberately small terminal chat client for OpenAI-compatible LLM endpoints.

The initial target is local models: one Linux binary, one config file, streaming chat, and as little context overhead as practical.

## Status

Factory bootstrap. No product implementation has started.

## Roadmap

- **Phase 0** — Factory bootstrap and contracts.
- **Phase 1** — single Go binary for Linux, config file, unauthenticated OpenAI-compatible chat.
- **Phase 2** — optional authentication for endpoints that require it.

Session persistence, databases, compaction, tools, agents, embeddings, and other harness features are explicitly deferred.

## Factory

This repository follows the Radilabs Factory structure from `naorw/core-concepts/skeleton`.

Read `PROJECT.md`, `PHASES.md`, and `TASKS.md` before implementation.
