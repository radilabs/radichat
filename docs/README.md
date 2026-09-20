# Documentation

`docs/` stores durable technical knowledge that future phases and future agents need.

Write to `docs/` when forgetting something could cause a future agent to make the wrong implementation choice.

Do not narrate implementation progress here. That belongs in the current phase task file.

## What Belongs Here

* External API behavior and constraints
* Normalized data schemas
* Caching and staleness rules
* Platform permission behavior
* Credential or configuration storage
* Build, install, and debugging procedures
* Design system rules and token conventions
* Known platform limitations

## What Does Not Belong Here

* Task execution progress (use the current phase task file)
* Decisions that future work must respect (use `decisions/`)
* Temporary findings that die with the current phase
