# Phase 5 adversarial test summary

Stage 3 / Phase 5 used the accepted `radichat-mock` to check RadiChat against the protocol and context behavior documented in [`protocol.md`](protocol.md), [`context-accounting.md`](context-accounting.md), and [`cli.md`](cli.md).

## Result

The bounded documented matrix found **zero confirmed divergences**. No Phase 5 finding or product fix was made. Zero findings is a valid outcome for this investigation.

The corrected matrix produced 83 verdict records:

| Verdict | Cases | Meaning |
| --- | ---: | --- |
| PASS | 63 | Documented expected behavior observed, including the harness negative control. |
| NOT RUN | 17 | Sixteen optional cases and one mock-supplied omitted-Content-Type subcase omitted with a recorded reason. |
| EXPLORATORY | 3 | Undocumented probes recorded as observations only; no defect oracle was claimed. |
| DIVERGENCE | 0 | No confirmed difference from documented behavior. |

The matrix covered:

- SSE success, fixed byte fragmentation, malformed events, and early termination;
- HTTP failures and non-SSE responses;
- authentication, token redaction, and redirect refusal;
- request-field assertions, context-budget boundaries, UTF-8 byte accounting, pair trimming, failed-turn rollback, `/clear`, and `max_tokens`.

Early calibration identified two test setup errors: the mock response had a sniffed `text/plain` content type, and two exact-fit budget values were one byte too small. Those candidate divergences were withdrawn. A separate early test skip also prevented a sibling subcase from running and was corrected. The final raw no-Content-Type response and corrected budget-boundary cases passed. No accepted product or mock implementation was changed.

Independent verification also passed. Luna checked five additional documented cases, and the Watcher independently reran the matrix, verified the source boundary and assertions, and checked the same five kinds of loopback cases with its own listener. Results were:

| Additional documented case | Independent observation |
| --- | --- |
| HTTP 204 with an empty body | Exit 1, one request, no retry. |
| 100 KiB HTTP 500, no auth | Exit 1, one request; diagnostic body snippet stayed within the documented 8 KiB body limit. |
| 100 KiB HTTP 500, authenticated | Exit 1, one request; Authorization was present and the untrusted response body marker was absent from diagnostics. |
| System prompt alone over budget | Exit 1 before network access; zero requests. |
| Input line over the configured limit | Luna: 5,001-byte line with budget 5,000. Watcher: 8,193-byte line with budget 8,192. Both failed locally with zero requests. |

Dr Watson's required harness corrections were completed and reviewed. These accepted summaries are recorded here so the public result does not depend on the raw reports or temporary scripts, which are not part of the public repository.

## Reproduce the committed matrix

Use Go 1.27.1 or a compatible later Go release from the repository root:

```sh
go version
go test -tags mockdiscovery ./internal/discovery -count=1 -run 'P5' -v
```

The command builds a temporary RadiChat binary, starts the in-process mock, and executes the tagged matrix. It creates disposable sanitized JSONL under `.factory/reports/`; that ignored output is not needed to interpret or reproduce the test result. The tracked test source and command are sufficient to rerun the matrix.

Additional repository checks used for the accepted result:

```sh
go test ./... -count=1
go vet ./...
go vet -tags mockdiscovery ./internal/discovery
go build ./...
gofmt -l .
git diff --check
```

Expected outcomes: test, vet, and build commands exit successfully; `gofmt -l .` prints no paths; `git diff --check` prints no errors.

## Bounded omissions and interpretation

Seventeen rows were marked NOT RUN: optional cases O1–O9 and U1–U7, plus the mock-supplied S6 omitted-Content-Type case because Go's HTTP server adds a sniffed type. The same documented behavior was exercised with a harness-local raw response that truly omitted the header and passed. Other omitted cases included large payloads, slow/idle behavior, and optional boundary probes. The three exploratory cases tested lone-CR framing, bytes after `[DONE]`, and a UTF-8 BOM. Their behavior is not asserted by the documented contract, so none is presented as a defect.

This investigation tested the documented OpenAI-compatible subset. It does not claim exact model-token accounting; RadiChat's context policy counts UTF-8 bytes and fixed overhead as described in [`context-accounting.md`](context-accounting.md).

## Phase and publication status

The Owner accepted Phase 5 on 2026-10-14 after independent Watcher PASS. No phase is currently authorized. The Build Week Discord post is a draft for Owner review and has not been published.
