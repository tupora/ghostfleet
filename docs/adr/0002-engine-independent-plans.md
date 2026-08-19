# ADR 0002: Engine-Independent Plans

Status: Accepted

## Context

The first online migration engine is gh-ost, but schema lifecycle semantics must
outlive a specific executable or database.

## Decision

Semantic schema changes, risk, dependencies, reversibility, and source and
destination fingerprints live in an engine-independent plan. An engine adapter
validates and executes the plan but does not define schema intent.

## Consequences

Adapters must declare supported operations and reject unsupported plans. Plan
formats require compatibility rules. Native MySQL and gh-ost behavior can be
tested against the same domain fixtures.

