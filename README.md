# RadiChat

RadiChat is a deliberately small terminal chat client for OpenAI-compatible LLM endpoints.

The initial target is local models: one Linux binary, one config file, streaming chat, and as little context overhead as practical.

## Status

Phase 1 (Local Chat Core) is accepted after independent verification. Phase 0 factory bootstrap is accepted. Phase 2 authentication is not authorized.

## Build (Linux)

Requires a Go toolchain. This tree was built with **Go 1.27.1** (`linux/amd64`); the module language version is **Go 1.22**. There are no third-party dependencies.

```bash
go build -o radichat .
```

That produces a single executable named `radichat` in the current directory.

```bash
go test ./...
go vet ./...
```

## Run

Copy the tracked example and edit it. The live file `radichat.json` is gitignored.

```bash
cp radichat.example.json radichat.json
./radichat
```

Or select another file:

```bash
./radichat -config /path/to/config.json
./radichat -help
```

The process reads only that JSON file for endpoint, model, context budget, generation reserve, and optional system prompt. No API key, environment login, or discovery call is used in Phase 1.

Type a line and press Enter to send it. Slash commands:

* `/clear` — drop in-memory conversation history; keep the configured system prompt
* `/quit` — exit

Blank lines are ignored. Unknown slash commands are rejected and not sent to the model. EOF on stdin exits. Ctrl-C cancels an in-flight request (discarding that turn) and exits.

## Documentation

* `docs/configuration.md` — config schema, URL rules, validation
* `docs/protocol.md` — HTTP/SSE subset, timeouts, limits
* `docs/context-accounting.md` — local byte-based working budget
* `docs/cli.md` — commands, output streams, exit codes, errors

## Roadmap

- **Phase 0** — Factory bootstrap and contracts (accepted).
- **Phase 1** — single Go binary for Linux, config file, unauthenticated OpenAI-compatible chat.
- **Phase 2** — optional authentication for endpoints that require it.

Session persistence, databases, compaction, tools, agents, embeddings, and other harness features are explicitly deferred.

## Factory

This repository follows the Radilabs Factory structure.

Read `PROJECT.md`, `PHASES.md`, and `TASKS.md` before implementation.
