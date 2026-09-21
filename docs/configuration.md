# Configuration

RadiChat reads a single JSON object. The default path is `./radichat.json` in the process working directory. Override with `-config PATH`.

A tracked example lives at `radichat.example.json`. Do not commit a live `radichat.json`.

No environment variables, API keys, or extra CLI flags are required for Phase 1.

## Fields

| Field | Required | Type | Meaning |
| --- | --- | --- | --- |
| `endpoint` | yes | string | Absolute `http://` or `https://` chat-completions URL. Sent exactly as written. |
| `model` | yes | nonempty string | Model name placed in the JSON `model` field. |
| `context_budget` | yes | positive integer | Application working-budget units for the whole request (see `docs/context-accounting.md`). |
| `generation_reserve` | yes | positive integer, strictly smaller than `context_budget` | Reserved units for the completion; also sent as `max_tokens`. |
| `system_prompt` | no | string | Optional. Omit the key, or set a JSON string. If the string is nonempty, it is sent as a fixed leading `system` message. JSON `null` and any other non-string value are rejected. |

Example:

```json
{
  "endpoint": "http://127.0.0.1:8080/v1/chat/completions",
  "model": "local-model",
  "context_budget": 8192,
  "generation_reserve": 1024,
  "system_prompt": "You are a concise assistant."
}
```

## Endpoint URL rules

* Scheme must be `http` or `https`.
* Host is required.
* Userinfo/credentials, query strings, and fragments are rejected.
* RadiChat does not append `/v1/chat/completions` or otherwise rewrite the path. If the server expects that path, put it in the URL.
* Authentication fields (`api_key`, `authorization`, and similar) are unknown keys and are rejected.

## Validation failures

The process exits nonzero and prints a short diagnostic to stderr when:

* the file is missing, unreadable, a directory, empty, or larger than 1 MiB;
* JSON is malformed, contains duplicate keys, unknown keys, trailing values, or wrong types (including `system_prompt` set to `null` or any non-string);
* budgets are missing, non-positive, non-integers, or `generation_reserve >= context_budget`;
* the endpoint URL is unusable under the rules above.

Errors name the problem and a corrective action. They do not dump the whole config file or request bodies.

A leading UTF-8 BOM is ignored. JSON comments are not allowed.
