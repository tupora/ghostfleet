# Architecture

GhostFleet separates desired-state reasoning from physical execution.

The schema path is introspection to normalized AST to fingerprint to semantic diff
to change IR. The migration path is risk analysis to dependency plan to engine
selection to immutable forward and rollback plans. The execution path is atomic
deployment claim to fenced task ownership to engine lifecycle to verified target
history.

## Boundaries

- Core schema and migration packages must not depend on a specific execution
  engine or storage implementation.
- PostgreSQL holds control-plane state; managed MySQL targets hold a minimal,
  append-only schema history used for reconciliation.
- The controller is stateless outside durable storage.
- Workers may act only while holding the current, unexpired fencing generation.
- Engines receive version-bound plans and cannot invent schema intent.

## Unresolved decisions

Control-plane reconciliation remains a Phase 0 storage decision. Target
credential handling, API authentication, supported compatibility ranges, engine
process isolation, and audit redaction are defined in
[ADR 0003](adr/0003-trust-boundaries-and-compatibility.md); changes to those
contracts require a new ADR.
