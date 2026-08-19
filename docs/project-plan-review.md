# Project Plan Review

## Decision

The project vision and safety model are strong enough to proceed. The roadmap
needs narrower release boundaries and explicit exit gates before implementation
scales beyond Phase 0.

## What is strong

- The product is framed as schema lifecycle management, not a gh-ost wrapper.
- The plan identifies the central distributed-systems hazards: duplicate claims,
  expired owners, stale workers, ambiguous control-plane state, and cutover.
- Desired and observed schemas are separated and compared through normalized
  representations and fingerprints.
- Forward and rollback planning are version-bound and occur before execution.
- Non-goals correctly reject transactional guarantees that MySQL cannot provide.

## Gaps to resolve

### The MVP is still too broad

The listed MVP spans introspection, semantic parsing, versioning, diffing,
planning, two execution engines, history, rollback planning, drift detection, and
an operator CLI. Treat Phases 1–4 as an MVP program with demonstrable increments,
not as one release. A v0.1 should prove deterministic import, diff, and planning
before any production DDL is enabled.

### Rollback cannot wait until Phase 7

Rollback execution may remain in Phase 7, but reversibility metadata, reverse
planning, immutable history, and data-loss classification are Phase 2–3 domain
requirements. Otherwise early formats and engine contracts will need unsafe
retrofits.

### Consistency contracts need to be explicit

The plan names PostgreSQL and target-side history but does not yet define which
store is authoritative during disagreement, what transaction boundaries are
required, how retries are identified, or how reconciliation is audited. These
belong in storage ADRs before migrations execute.

### Security and compatibility are underspecified

Phase 0 must define credential handling, target authorization, audit redaction,
supported MySQL and gh-ost versions, and the policy for running externally
supplied engine commands. Multi-tenancy can remain a non-goal, but the trust
boundary cannot.

### Operational readiness needs exit gates

Every execution phase needs measurable recovery objectives, fault-injection
tests, upgrade compatibility, operator runbooks, and observable state transitions.
Unit coverage alone is not a sufficient safety gate.

## Recommended release boundaries

- Foundation: Phase 0, with buildable processes and accepted architecture records.
- v0.1 planning preview: Phases 1–3, read-only against target databases.
- v0.2 single-target alpha: Phase 4, native DDL before gh-ost is enabled.
- v0.3 fleet preview: Phases 5–7, including fencing and explicit rollback policy.
- v0.4 operations preview: Phase 8.
- v1.0 candidate: Phase 9 plus security, compatibility, and recovery gates.

## Cross-cutting requirements

Each issue that changes behavior must identify:

- the safety invariant it preserves;
- the failure mode it introduces or changes;
- the persisted and API compatibility impact;
- the deterministic test or fault scenario that proves acceptance;
- whether the operation is read-only, reversible, destructive, or unsupported.

