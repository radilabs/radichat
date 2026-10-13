# Factory v1.2 Migration

**Status:** Accepted (with Phase 3 owner acceptance)

## Context

radichat was bootstrapped on the pre-v1.2 Factory skeleton. Core Concepts v1.2 added the Architect and Writer/Documentation Steward role contracts, the ephemeral `.factory/` materialization model (core-concepts decision 0005), and a public-surface check that forbids tracked Factory runtime paths. Before this migration the v1.2 check failed against radichat: root `DRIVER.md`, `WATCHER.md`, and `reports/README.md` were tracked in a public repository.

## Decision

0. **Pin rule.** Core Concepts is pinned by **exact commit SHA** (40 hex characters) — never a branch, tag, or other moving ref. The pin for this migration is `12d4682c9e6497d03a0ba5a94652a6fc200ab1af`. Rehydration checks out that exact SHA.
1. radichat adopts Factory Core Concepts v1.2. Before delegated runs, the Driver materializes the ephemeral kit from the pinned commit:

   ```sh
   git -C <core-concepts> checkout 12d4682c9e6497d03a0ba5a94652a6fc200ab1af
   sh <core-concepts>/skeleton/materialize.sh . runtime
   ```

   The pin is recorded in `.factory/source.txt` (ephemeral) and authoritatively in this decision and the Phase 3 handoff (persistent).
2. Role entrypoints (Driver, Architect, Coder, Writer, Watcher) live in the disposable `.factory/` kit. Root `DRIVER.md` and `WATCHER.md` are no longer tracked; their content is superseded by the canonical contracts plus `execution-engine.md` (CAO qualification, bounded recovery, transport evidence) and the `agents.yaml` bindings.
3. Runtime reviewer/Watcher reports are written under `.factory/reports/`. The historical `reports/` directory remains local-only, gitignored evidence for accepted phases.
4. Publication rule: `.factory/`, `reports/`, root `DRIVER.md`, root `WATCHER.md`, and `factory-kit/` must never be tracked. `sh .factory/check-public-surface.sh` must pass before any release/container publication.
5. `findings/` is the durable defect inbox. A finding is evidence, not an authorized task.

## Consequences

- Factory internals can be upgraded centrally by re-materializing from a newer pinned Core Concepts version; record the version change when it could materially affect results.
- A fresh clone has no `.factory/`; rehydrate from the recorded Core Concepts commit before delegated work.
- Changing the kit version is an execution-environment change; it does not by itself change project lifecycle semantics.
