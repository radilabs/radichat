# Documentation

This directory contains RadiChat's durable user and protocol documentation.

- [Configuration](configuration.md) — JSON fields, endpoint rules, optional authentication, and validation failures.
- [Protocol subset](protocol.md) — request/stream format, transport behavior, limits, and incompatible endpoints.
- [Context accounting](context-accounting.md) — deterministic working-budget and history-trimming policy.
- [CLI behavior](cli.md) — invocation, commands, output streams, and exit codes.
- [Batch input and JSONL results](batch.md) — file input, JSONL v1, output-file rules, and signals.
- [Kubernetes Job example](kubernetes.md) — structural Job/ConfigMap example. No cluster or image run is claimed.
- [Container image](container.md) — public image references, configuration mounts, networking, and authentication.
- [Testing with radichat-mock](testing-mock.md) — development mock of the endpoint for protocol and auth tests; not part of the product.
- [Phase 5 investigation summary](testing-phase-5.md) — public test coverage, reproducible commands, results, and limits from the bounded adversarial investigation.
- [Accepted handoffs](handoffs/README.md) — concise snapshots of verified phase checkpoints, including Phases 4 and 5.

The repository root [README](../README.md) is the public starting point for installation and first use.
