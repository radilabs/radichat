# RadiChat

RadiChat is a deliberately small terminal chat client for OpenAI-compatible LLM endpoints.

The initial target is local models: one Linux binary, one config file, streaming chat, and as little context overhead as practical.

## Status

Phase 2 (Endpoint Authentication) is accepted after owner testing and independent verification. Phase 1 (Local Chat Core) and Phase 0 factory bootstrap are also accepted. Optional bearer authentication is available for endpoints that require it; local unauthenticated use is unchanged.

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

The process reads that JSON file for endpoint, model, context budget, generation reserve, optional system prompt, and optional `bearer_token_env`. Local use does not require a token. When `bearer_token_env` is set, RadiChat reads the named process environment variable at startup and sends `Authorization: Bearer <token>` to the configured endpoint only. The config file must contain the variable name, never the token.

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
- **Phase 1** — single Go binary for Linux, config file, unauthenticated OpenAI-compatible chat (accepted).
- **Phase 2** — optional bearer authentication via an environment variable named in the config file.

Session persistence, databases, compaction, tools, agents, embeddings, and other harness features are explicitly deferred.

## Factory

This repository follows the Radilabs Factory structure.

Read `PROJECT.md`, `PHASES.md`, and `TASKS.md` before implementation.
