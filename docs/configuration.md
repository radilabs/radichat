# Configuration

RadiChat reads a single JSON object. The default path is `./radichat.json` in the process working directory. Override with `-config PATH`.

A tracked example lives at `radichat.example.json`. Do not commit a live `radichat.json`.

Local unauthenticated use needs no environment variables. Optional bearer authentication names a process environment variable in the config file; the token itself is never stored in JSON.

## Fields

| Field | Required | Type | Meaning |
| --- | --- | --- | --- |
| `endpoint` | yes | string | Absolute `http://` or `https://` chat-completions URL. Sent exactly as written. |
| `model` | yes | nonempty string | Model name placed in the JSON `model` field. |
| `context_budget` | yes | positive integer | Application working-budget units for the whole request (see `docs/context-accounting.md`). |
| `generation_reserve` | yes | positive integer, strictly smaller than `context_budget` | Reserved units for the completion; also sent as `max_tokens`. |
| `system_prompt` | no | string | Optional. Omit the key, or set a JSON string. If the string is nonempty, it is sent as a fixed leading `system` message. JSON `null` and any other non-string value are rejected. Mutually exclusive with `system_prompt_file`. |
| `system_prompt_file` | no | nonempty string | Optional. Path of a UTF-8 file whose contents become the system prompt. Relative paths resolve against the directory of the config path as written. Absolute paths are used directly. Mutually exclusive with `system_prompt`, including when the inline value is empty. |
| `bearer_token_env` | no | nonempty string | Optional. Omit the key for the zero-auth local path. When present, it must be a POSIX environment variable name (letters, digits, underscore; not starting with a digit). RadiChat reads that variable at startup. JSON `null` and any other non-string value are rejected. |

Example (local, no authentication):

```json
{
  "endpoint": "http://127.0.0.1:8080/v1/chat/completions",
  "model": "local-model",
  "context_budget": 8192,
  "generation_reserve": 1024,
  "system_prompt": "You are a concise assistant."
}
```

Optional authenticated mode names the environment variable only:

```json
{
  "endpoint": "https://example.invalid/v1/chat/completions",
  "model": "remote-model",
  "context_budget": 8192,
  "generation_reserve": 1024,
  "bearer_token_env": "RADICHAT_BEARER_TOKEN"
}
```

Export the token in the process environment before starting RadiChat. Do not put the token in the JSON file, command line, or example configs.

## System prompt file

`system_prompt_file` is read once at startup by `Load`. `Parse` checks the string and the conflict with `system_prompt` but does not open the file.

The file must be a regular file of at most 1 MiB after symlink resolution. FIFOs and directories are rejected. The bytes must be valid UTF-8. One leading UTF-8 BOM is removed, and one trailing LF or CRLF is removed. The result must be non-empty. A prompt that still does not fit `context_budget` with an empty user message and `generation_reserve` fails startup before any request. Errors include the path and the problem, not the prompt text.

An omitted system prompt, or `"system_prompt": ""` without `system_prompt_file`, does not add that startup fit check.

## Endpoint URL rules

* Scheme must be `http` or `https`.
* Host is required.
* Userinfo/credentials, query strings, and fragments are rejected.
* RadiChat does not append `/v1/chat/completions` or otherwise rewrite the path. If the server expects that path, put it in the URL.
* Direct token fields (`api_key`, `authorization`, and similar) are unknown keys and are rejected. Use `bearer_token_env` to name an environment variable instead of embedding a secret.

## Authentication

When `bearer_token_env` is omitted, RadiChat sends the same request headers as Phase 1 and does not set `Authorization`.

When `bearer_token_env` is present:

* the name is validated before any network access;
* the named variable must be set to a nonempty value that is a single-line HTTP header token (printable ASCII 0x21–0x7E, including `~`; no space or control characters, including CR/LF);
* missing, empty, or malformed values fail at startup with a diagnostic that names the problem and the variable, never the token;
* each request to the configured endpoint includes exactly `Authorization: Bearer <token>`;
* the token stays in process memory for the lifetime of the process and is not written to disk, logs, or error text;
* HTTP and stream error diagnostics keep the status or classification and a fix hint, but omit untrusted server response bodies;
* transport failures are classified without quoting the underlying error or redirect Location;
* streamed assistant text is filtered so a bearer token is not printed, including across split SSE deltas and when a successful stream ends on a proper prefix of the token. Ordinary surrounding text is kept. Unauthenticated diagnostics and stream text are unchanged.

The process environment is not encrypted. Other processes running as the same user can still inspect it.

## Validation failures

The process exits nonzero and prints a short diagnostic to stderr when:

* the file is missing, unreadable, a directory, empty, or larger than 1 MiB;
* JSON is malformed, contains duplicate keys, unknown keys, trailing values, or wrong types (including `system_prompt`, `system_prompt_file`, or `bearer_token_env` set to `null` or any non-string);
* both `system_prompt` and `system_prompt_file` are present, or the prompt file is missing, not a regular file, too large, not UTF-8, or empty after the single BOM and newline strip;
* a non-empty system prompt does not fit the context budget at startup;
* budgets are missing, non-positive, non-integers, or `generation_reserve >= context_budget`;
* the endpoint URL is unusable under the rules above;
* `bearer_token_env` is present but the name is invalid, the variable is unset or empty, or the value cannot be sent as an HTTP header.

Errors name the problem and a corrective action. They do not dump the whole config file, request bodies, or secret values.

A leading UTF-8 BOM is ignored. JSON comments are not allowed.
