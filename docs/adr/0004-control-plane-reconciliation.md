# ADR 0004: Control-Plane Persistence and Reconciliation

Status: Accepted

Review record: [Issue #3](https://github.com/tupora/ghostfleet/issues/3)

## Context

The control plane needs durable records for desired versions, deployments, tasks,
leases, fencing generations, immutable plans, and audit events. A read followed by
an insert or update is unsafe: two controllers can observe the same state and
both claim work. The target's append-only history is also an observation, not a
replacement for control-plane intent.

## Decision

PostgreSQL is authoritative for desired versions, deployment intent, task state,
lease ownership, plan bytes, and audit history. The managed MySQL history is
authoritative only for what the target reports as applied. Reconciliation treats
an unknown target observation, an unknown target version, a fingerprint mismatch,
or a control-plane/target disagreement as `PAUSE`; it never repairs either side
automatically or schedules a new mutation from ambiguous state.

The schema in
[`storage/migrations/001_control_plane.sql`](../../storage/migrations/001_control_plane.sql)
models:

- `schema_versions`: immutable desired versions keyed by target and version ID;
- `deployments`: one immutable intent per target/version pair;
- `tasks`: retryable work state and the current fencing generation;
- `leases`: append-only ownership records for each generation;
- `plans`: versioned, fingerprint-bound plan bytes;
- `target_history`: observed applied versions and fingerprints;
- `audit_events`: append-only state-transition evidence.

Uniqueness is enforced by database keys, not caller-side checks. In particular,
`(target_id, version_id)` is unique for deployments and target history, and a
lease is unique for `(task_id, generation)`. Immutable tables reject updates and
deletes through database triggers.

## Atomic transactions and retries

Version creation and task claims are conditional writes inside one transaction.
The claim transaction locks the candidate task, checks that the requested
generation is newer than the stored generation and that the previous lease is
expired, writes the new lease and task owner, and commits both records together.
A stale generation returns `STALE_FENCE`; an unexpired owner returns `CONFLICT`.

Every retry carries an idempotency key. The same key and request payload returns
the recorded result; the same key with a different target, version, or plan
fingerprint returns `CONFLICT`. A transaction failure rolls back the lease,
owner, audit event, and state transition together, so a retry can safely re-enter
the claim path.

## Recovery behavior

| Observation | Decision | Reason |
| --- | --- | --- |
| Control plane and target history match | Continue | Both intent and applied evidence are known. |
| Target history is missing or unreadable | Pause | INV-007 and INV-010 prevent speculative work. |
| Target reports an unknown version | Pause | The target may have been changed outside GhostFleet. |
| Fingerprints disagree | Pause | A version ID alone is not proof of identical schema. |
| A worker lease expires | Allow a newer generation to claim | The old generation cannot mutate state after fencing. |
| A stale worker retries | Reject with `STALE_FENCE` | INV-003 prevents stale ownership from changing state. |

Recovery is operator-directed: inspect target state, record a new reconciliation
event, and only then create a new desired version or re-enter a paused task.
Target history is never silently overwritten.

## Consequences and compatibility

The schema grows by additive, numbered migrations and preserves completed
deployment, plan, target-history, and audit rows. Writers emit the current format
and readers accept the current and immediately previous plan format, as required
by [ADR 0003](0003-trust-boundaries-and-compatibility.md). Conditional writes and
database constraints make concurrency behavior inspectable, while the explicit
pause decisions reduce availability during uncertainty.

Rejected alternatives: a read-then-write claim race, a mutable “latest state”
row without an audit trail, and treating target history as a desired-state
source would each permit duplicate or unverified execution.

