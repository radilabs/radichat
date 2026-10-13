# Operator Findings

This is the durable inbox for defects and unexpected behavior observed during real runs, including operator, user, and independent verifier observations.

Findings are **evidence, not authorized tasks**. Recording a finding does not expand the active phase, permit a worker to implement a fix, or supersede the project owner’s acceptance gate.

## Naming and capture

Create one Markdown file per distinct finding: `F-0001-short-title.md`, with a stable unique ID. A report may reference several findings; do not duplicate the same underlying defect.

Each finding records:

- **ID / title** and **status** (`new`, `triaged`, `converted`, or `closed`);
- **source** (operator/Watcher/user/etc.), **date**, project phase and run/session reference;
- **expected** and **observed** behavior;
- **environment/context** and reproduction steps, if known;
- **evidence** (paths or links to retained logs, traces, screenshots, test failures; redact secrets);
- **impact/severity** and uncertainty, without pretending unverified claims are proven;
- **triage result** (confirmed bug, duplicate, expected behavior, insufficient evidence, or other);
- **linked task(s)** or decision, when applicable;
- **closure**: disposition, verification/regression protection and date.

Unknown information may be marked `unknown`. Do not block capture just because reproduction is incomplete.

## Lifecycle

1. **Capture:** operator records observations promptly, preserving inspectable evidence. Do not copy large raw logs or sensitive material into Git.
2. **Triage:** authorized reviewer investigates, deduplicates and classifies. A report alone is not a PASS/FAIL verdict.
3. **Disposition:** confirm and create/link one or more fix tasks through the normal owner-authorized task/phase process, or close as duplicate/expected/not actionable with rationale. The finding stays in place.
4. **Verify and close:** after authorized implementation, link verification evidence and regression protection where practical. Close only when the disposition is recorded.

Do not move findings into `tasks/` or erase them when converted; keep stable backlinks in both directions.

Ephemeral verifier reports live under `.factory/reports/`; persistent defects live here. Promote reusable technical knowledge into `docs/` and decisions into `decisions/`, rather than using findings as general project documentation.
