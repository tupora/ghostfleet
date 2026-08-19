# ADR 0003: Trust Boundaries and Compatibility Policy

Status: Accepted

Review record: [Issue #2](https://github.com/tupora/ghostfleet/issues/2)

## Context

GhostFleet coordinates privileged database changes across an API, a controller,
workers, the PostgreSQL control plane, managed MySQL targets, and an optional
`gh-ost` subprocess. A credential or compatibility decision made at one boundary
must not be inferred by another component. The Phase 0 support range is
deliberately narrow so an unknown runtime or persisted format fails closed.

## Trust boundaries and authentication

The API is the only public control-plane entry point. API callers authenticate
with mutually authenticated TLS and an operator identity; authorization is
evaluated for each target and operation. The controller authenticates to
PostgreSQL with a dedicated service identity and authenticates to workers with
mutual TLS. A worker authenticates to a target with a target-scoped credential
reference and never receives API credentials or another target's credentials.

The controller, worker, and engine are separate trust zones:

```text
operator/API client --mTLS + request identity--> API
                                                  |
                                      target-scoped authorization
                                                  v
                                               controller
                                           /                 \
                         mTLS + worker identity             PostgreSQL service identity
                                     v                         v
                                  worker              control-plane database
                                     |
                           target credential reference
                                     v
                              managed MySQL target
```

Liveness, a network address, or possession of a bearer token is not proof of
ownership. A request with an unknown identity, target, generation, plan
fingerprint, or idempotency key is rejected with a stable application error and
does not change state. Retries reuse the same idempotency key; a completed
operation returns its recorded result and a conflicting payload is a conflict.

## Credential storage, transport, rotation, and audit redaction

- Development may read DSNs from environment variables; production stores only
  a secret-manager or platform-secret reference in GhostFleet configuration.
- Secret values are supplied to the process through the secret manager or a
  protected file descriptor and are never persisted in plans, API messages,
  PostgreSQL rows, fixtures, command-line arguments, or logs.
- TLS is required for API, control-plane, and target connections outside local
  development. Certificate and target credential validation occurs before a
  plan can be claimed.
- Rotation creates a new credential version, verifies it with a read-only
  connection, then retires the previous version after in-flight work drains.
  A failed verification leaves the old version active and records only a
  redacted reason.
- Audit events contain actor, target, operation, plan fingerprint, outcome,
  timestamps, and credential version identifiers, never credential material.
  Redaction removes DSN passwords, authorization headers, private keys, and
  secret-looking values before structured logging or artifact collection.

Rejected alternatives: storing raw credentials in the control plane would make
the database a secret vault without the required rotation and access controls;
passing credentials in command-line arguments would expose them through process
inspection; and logging full connection strings would leak secrets through CI
artifacts. None is permitted.

## Supported compatibility range

Phase 0 supports the following exact compatibility window:

| Component | Supported range | Policy |
| --- | --- | --- |
| Go toolchain | `>=1.26.0, <1.27.0` | The module and CI use Go 1.26; a new minor range requires a compatibility review. |
| MySQL target | `8.4.x` | Native DDL and target-history behavior are tested only against the 8.4 family. |
| PostgreSQL control plane | `17.x` | Migrations and transaction semantics are tested only against PostgreSQL 17. |
| gh-ost | `v1.1.10` | The binary is pinned by release, checksum-verified, and invoked only after capability validation. |

The gh-ost release is pinned to a production release rather than an arbitrary
branch; see the [gh-ost releases](https://github.com/github/gh-ost/releases).
Supporting another range requires an ADR, fixture coverage, and an explicit
compatibility entry. Unknown versions pause before any database mutation.

## Engine subprocess isolation and command construction

The engine adapter starts an allowlisted, checksum-verified executable with
`exec.CommandContext` and an argument vector. It never invokes a shell, parses a
user-provided command string, interpolates SQL into an argument template, or
inherits the caller's environment wholesale. The adapter supplies a minimal
environment containing only the non-secret locale and required binary settings;
credentials use a protected input channel. Context cancellation terminates the
process and its descendants, and the adapter records exit status without
including arguments that may contain sensitive values.

Only the engine adapter may construct the command. The plan supplies validated
structured fields; it cannot select an executable path, add flags, or override
the target identity. Unsupported flags, malformed identifiers, and a checksum
or version mismatch are typed rejection errors. The first implementation keeps
the adapter boundary ready but does not enable production database mutation.

Rejected alternatives: `sh -c` permits shell expansion and command injection;
accepting an arbitrary executable path defeats the allowlist; and passing a DSN
as a flag exposes credentials to process listings.

## Upgrade and persisted-format compatibility

- Database migrations are forward-only, numbered, checksum-verified, and
  expand/contract: additive columns and readers land before writers, and removal
  waits for all supported readers to stop using the old field.
- Deployment history, audit events, plans, and schema fingerprints are
  append-only. Completed history is immutable; correction is a new event, never
  an update in place.
- Every serialized plan carries a format version, canonical field ordering, and
  source/destination fingerprints. Readers accept the current and immediately
  previous format; writers emit only the current format. Unknown versions pause
  before execution.
- API packages are versioned (`ghostfleet.v1`). Additive protobuf fields are
  allowed; field numbers and meanings are never reused. Removing or changing a
  field, error code, or idempotency semantic requires a new API version and an
  explicit migration note.
- An upgrade must be restartable and must preserve leases, generations, audit
  history, plan bytes, and target reconciliation evidence. A failed migration
  blocks rollout and leaves the last known compatible reader active.

## Consequences

The supported matrix is narrower than the underlying tools, and credential
rotation requires an external secret-management capability. These costs buy
deterministic CI, reviewable upgrades, safe redaction, and an unambiguous pause
when ownership, compatibility, or evidence is incomplete.

