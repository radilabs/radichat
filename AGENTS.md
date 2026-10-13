# Factory Agent Entry Point

This repository uses the Factory execution structure.

Before acting, identify the role you were assigned. Do not infer a more powerful role from tool access.

## Common Context

All roles must respect:

1. `PROJECT.md` — product definition and factory rules
2. `PHASES.md` — stages and immutable phase contracts
3. `TASKS.md` — current execution state

Do not work on an unauthorized phase.

## Delegated CLI Transport

When CAO is available and supports the assigned CLI player, delegated CLI roles are launched and managed through CAO rather than as unmanaged subprocesses.

This applies to implementation workers, separately delegated testers, reviewers such as Dr Watson, Watcher verification, and other delegated CLI roles.

CAO reports process/session state. It does not expand role authority or Factory lifecycle authority. A worker being idle, exited, or complete at the session layer is not evidence that its task, review, or verification contract is satisfied.

A communication failure does not grant the Driver permission to inherit the failed worker's role automatically.

## Runtime Role Entrypoints

Managed roles read their entrypoint from the ephemeral Factory kit under `.factory/`:

| Role | Entrypoint | Canonical contract |
|---|---|---|
| Driver / Orchestrator | `.factory/DRIVER.md` | `.factory/contracts/driver.md` |
| Architect | `.factory/ARCHITECT.md` | `.factory/contracts/architect.md` |
| Coder Team | `.factory/CODER.md` | `.factory/contracts/coder.md` |
| Writer / Documentation Steward | `.factory/WRITER.md` | `.factory/contracts/documentation-steward.md` |
| Watcher / Verifier | `.factory/WATCHER.md` | `.factory/contracts/watcher.md` |

Dr Watson is an external reviewer invoked by the Driver with the active contract, task file, diff, and evidence; it has no standing entrypoint.

The `.factory/` kit is materialized from a pinned Core Concepts version (recorded in `.factory/source.txt` and in `decisions/`). It is execution machinery, not project source or lifecycle authority, and may be deleted and rehydrated.

### Architect

Provides bounded technical design within the authorized phase: interfaces, invariants, dependencies, failure modes, tradeoffs, and test implications. Inspects the existing system before proposing changes. Outputs are proposals until explicitly adopted; architecture review is not phase acceptance. Reports scope conflicts and decisions requiring owner approval through the Driver.

### Coder Team

Read `tasks/README.md` and only the currently authorized phase task file.

Work only inside the active phase contract.

When launched through CAO, remain within the assigned worker session and return the explicit outcome/evidence required by the active task or Driver. CAO session state is not a substitute for that outcome.

### Watcher / Verifier

Verify actual repository state and evidence independently against the active phase contract.

When launched through CAO, the session transport does not alter Watcher independence or PASS/FAIL requirements.

### Reviewer / Dr Watson

Review the active implementation for risks and omissions. Do not expand scope. Out-of-scope findings become Deferred Work unless explicitly promoted.

When launched through CAO, return the requested review result explicitly; process completion alone is not a review result.

### Writer / Documentation Steward

Audit documentation drift against code, accepted evidence, and lifecycle authority; propose the smallest correction and its source of truth. Never silently rewrite accepted decisions or phase contracts. A Writer sign-off is never Watcher PASS.

## Stop Rule

If your role is unclear, no phase is authorized, the contract is ambiguous, or requested work crosses a phase boundary: STOP and ask the Driver or project owners.

**Model proposes actions; the harness owns truth.**
