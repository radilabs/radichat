# Decision Records

`decisions/` stores architectural and product decisions that future work must respect.

Do not create decision records merely to narrate implementation.

Do not silently rewrite accepted historical records. Supersede them with a new record.

## Decision Record Format

### [DECISION TITLE]

**Status:** [Proposed | Accepted | Superseded]

**Context:**
[What is the issue or situation requiring a decision?]

**Decision:**
[What was decided?]

**Consequences:**
[What becomes easier or harder because of this decision?]

---

## File Naming

Use a sequential prefix: `0001-`, `0002-`, etc.

Use kebab-case for the title: `0001-application-identity.md`.

Keep titles specific and searchable.
