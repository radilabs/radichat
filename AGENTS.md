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

## Role Entry Points

### Driver / Orchestrator

Read `DRIVER.md`, then `tasks/README.md` and the currently authorized phase task file.

The Driver coordinates execution and gates. It does not invent work or accept phases.

### Coder Team

Read `tasks/README.md` and only the currently authorized phase task file.

Work only inside the active phase contract.

When launched through CAO, remain within the assigned worker session and return the explicit outcome/evidence required by the active task or Driver. CAO session state is not a substitute for that outcome.

### Watcher / Verifier

Read `WATCHER.md`, the active phase contract in `PHASES.md`, and the active phase task file.

Verify actual repository state and evidence independently.

When launched through CAO, the session transport does not alter Watcher independence or PASS/FAIL requirements.

### Reviewer / Dr Watson

Review the active implementation for risks and omissions. Do not expand scope. Out-of-scope findings become Deferred Work unless explicitly promoted.

When launched through CAO, return the requested review result explicitly; process completion alone is not a review result.

## Stop Rule

If your role is unclear, no phase is authorized, the contract is ambiguous, or requested work crosses a phase boundary: STOP and ask the Driver or project owners.

**Model proposes actions; the harness owns truth.**
