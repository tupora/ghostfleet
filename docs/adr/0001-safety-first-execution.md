# ADR 0001: Safety-First Execution

Status: Accepted

## Context

DDL is often irreversible at the data layer, while distributed ownership and
database observations can become ambiguous.

## Decision

GhostFleet fails closed. Unknown drift, stale fencing generations, mismatched plan
fingerprints, inconsistent history, or insufficient control-plane state pause or
reject execution. No component may infer ownership from liveness alone.

## Consequences

Availability may be reduced during partitions and recovery. Operators receive an
explicit state requiring reconciliation instead of speculative DDL. Tests must
exercise rejected stale-owner and uncertain-state transitions.

