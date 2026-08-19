# Failure Model and Safety Invariants

GhostFleet pauses when it cannot prove ownership, target state, or plan validity.

## Invariants

- INV-001: only one valid worker owns a migration task at a time.
- INV-002: an applied schema version never executes again implicitly.
- INV-003: a worker with an outdated generation cannot change migration state.
- INV-004: unexpected drift blocks migration by default.
- INV-005: completed deployment history is immutable.
- INV-006: data-destructive rollback is never classified as safe.
- INV-007: insufficient control-plane certainty pauses execution.
- INV-008: only the current migration owner may initiate cutover.
- INV-009: every plan is bound to explicit source and destination fingerprints.
- INV-010: observed production state is verified before apply or rollback.

## Required fault scenarios

Tests must cover controller and worker crashes, network partitions, lease
expiration, stale-worker recovery, duplicate requests, duplicate scheduling,
database disconnects, engine crashes, and loss during cutover preparation.

The critical fencing proof is: worker A loses its lease, worker B acquires the
next generation, worker A reconnects, and every state-changing operation from
worker A is rejected.

