# Roadmap

The GitHub milestones and project are the operational source of truth. This file
records phase boundaries and exit outcomes.

## Phase 0 — Foundation

Buildable controller, worker, and CLI processes; configuration and structured
logging; error and storage contracts; versioned API definitions; local MySQL and
PostgreSQL services; CI; testing conventions; architecture and security records.

Exit gate: a clean checkout passes make check and the unresolved trust,
consistency, and compatibility decisions have owners.

## Phase 1 — Schema Foundation

MySQL introspection, normalized AST, deterministic fingerprints, import and
baseline creation, registry history, and pull, inspect, and validate CLI commands.

Exit gate: repeated import of an unchanged database produces byte-identical
artifacts and the same fingerprint without executing DDL.

## Phase 2 — Diff and Versioning

Semantic schema diff, change IR, version graph and checksums, atomic duplicate
prevention, target-side history, drift detection, and operator CLI commands.

Exit gate: GhostFleet reliably classifies a target as in sync, drifted, or unknown
and never schedules work for an already-applied version.

## Phase 3 — Migration Planning

Dependency planning, risk policy, engine selection, forward and rollback plans,
reversibility classification, and plan CLI commands.

Exit gate: every plan is deterministic, bound to source and destination
fingerprints, and includes explicit rollback and data-loss implications.

## Phase 4 — Single-Target Execution

Native MySQL and gh-ost adapters, migration state machine, immutable history,
pause and resume, controlled cutover, recovery, and apply and migration CLI
commands.

Exit gate: the supported end-to-end workflow succeeds on one target under tested
crash and retry scenarios without duplicate DDL.

## Phase 5 — Distributed Execution

Worker service, scheduler, leases, fencing generations, heartbeats, atomic task
claims, failover, concurrency controls, and partition tests.

Exit gate: a stale worker cannot perform any state-changing action after a newer
generation takes ownership.

## Phase 6 — Rollout and Cutover

Rolling and canary strategies, observation windows, health gates, distributed
cutover barrier, manual approval, and automatic rollout halt.

Exit gate: unhealthy canaries stop later batches and every cutover is attributable
to the current owner and an explicit release decision.

## Phase 7 — Rollback

Version and migration rollback execution, revalidation, approval for destructive
operations, rollback history, and reapply semantics.

Exit gate: supported rollbacks are deterministic and auditable; destructive or
data-irrecoverable operations can never be classified as safe.

## Phase 8 — Global Safety

Fleet telemetry, resource budgets, adaptive concurrency, replication-lag policy,
production health gates, runbooks, and overload tests.

Exit gate: fleet pressure can halt or reduce migration work without surrendering
ownership safety or losing resumability.

## Phase 9 — GitOps and Ecosystem

GitHub Action and PR plans, approval policy, Go client, Kubernetes and Helm
packaging, dashboards, release automation, compatibility policy, and operator
documentation.

Exit gate: a supported release can be installed, upgraded, observed, audited, and
operated through the documented GitOps workflow.

