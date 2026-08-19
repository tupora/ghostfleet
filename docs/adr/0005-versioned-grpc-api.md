# ADR 0005: Versioned gRPC API and Generation

Status: Accepted

Review record: [Issue #4](https://github.com/tupora/ghostfleet/issues/4)

## Context

The API must expose stable Phase 0 boundaries without coupling callers to
storage or an execution engine. Generated clients and server interfaces must be
reproducible from the checked-in protobuf source, and a compatible change must
be distinguishable from a wire-breaking change before review.

## Decision

The `ghostfleet.v1` package exposes two boundaries:

- `SchemaService` reads version identity, parentage, checksums, fingerprints,
  and target association.
- `MigrationService` reads migration intent and claims a migration with an
  idempotency key and expected fencing generation.

The service returns domain data only; it does not expose database handles,
engine command lines, secret values, or mutable internal records. The API is
read-only unless the caller supplies an authenticated claim request. A claim is
accepted only when the control plane can atomically fence the requested
generation.

## Authentication, metadata, and error mapping

API callers use mutual TLS. The caller identity, request ID, idempotency key, and
deadline travel in gRPC metadata; idempotency keys are not inferred from a
connection or request ordering. The server validates target authorization before
loading or claiming a migration and redacts metadata before logging.

Application errors map to canonical gRPC status codes while retaining the stable
`ErrorCode` value in the error details:

| Application code | gRPC status | Retry rule |
| --- | --- | --- |
| `CONFLICT` | `ALREADY_EXISTS` | Do not retry with the same payload; reuse the idempotency key only to retrieve its result. |
| `DRIFT_DETECTED` | `FAILED_PRECONDITION` | Pause and reconcile target history. |
| `FAILED_PRECONDITION` | `FAILED_PRECONDITION` | Correct the request or configuration first. |
| `NOT_FOUND` | `NOT_FOUND` | Do not retry until the resource exists. |
| `STALE_FENCE` | `ABORTED` | Refresh ownership and claim a newer generation. |
| `VERSION_ALREADY_APPLIED` | `ALREADY_EXISTS` | Return recorded history; never execute implicitly. |
| `INTERNAL` | `INTERNAL` | Retry only with the same idempotency key after checking recorded state. |

Unknown error codes, missing metadata required by the operation, incompatible
API versions, and ambiguous authorization fail closed with no state change.

## Reproducible generation and compatibility checks

`api/proto` is the source of truth. `buf.yaml` pins lint and breaking-change
policy, while `Makefile` pins the Go and gRPC generator versions used by
`buf.gen.yaml`. `make api-generate` installs those generators into the ignored
local tools directory and regenerates the committed Go files under `api/gen`.
`make api-check` lints the source, regenerates the output, verifies that
generation is clean, and compares the schema with the `main` branch using Buf's
`FILE` breaking rules.

Compatibility rules are additive within `v1`: field numbers and meanings are
never reused, existing RPCs and message fields are not removed or repurposed,
and a changed meaning requires `ghostfleet.v2` plus a migration note. Generated
files are checked in so a clean checkout builds without a code-generation tool.

Rejected alternatives: hand-maintained Go DTOs would allow wire drift; generating
only in CI would hide local incompatibilities; and accepting arbitrary breaking
changes in `v1` would make persisted plans and clients unsafe to upgrade.

## Consequences

Buf and the pinned generators are build-time dependencies, while runtime code
depends only on the generated protobuf and gRPC packages. The API remains
narrow until storage and execution semantics are ready, and every schema change
has an inspectable generated artifact and compatibility result.