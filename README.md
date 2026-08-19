# GhostFleet

GhostFleet is a distributed database schema lifecycle manager for safe, versioned,
reversible, and auditable database changes. The first implementation targets MySQL
and supports native DDL and gh-ost behind an engine-neutral planning model.

> [!WARNING]
> GhostFleet is pre-alpha. It must not be used against production databases yet.

## Design priorities

- Stop when ownership or database state is ambiguous.
- Treat desired schema as the source of truth.
- Bind every plan to explicit source and destination fingerprints.
- Make version claims, task ownership, and worker fencing atomic.
- Preserve immutable deployment and rollback history.
- Separate schema reversibility from data recoverability.

## Repository layout

- cmd: controller, worker, and CLI entry points
- internal: application foundations and control-plane internals
- schema: schema AST, introspection, normalization, diff, and drift
- migration: planning, policy, rollback, and state machine
- engine: execution-engine contracts and adapters
- storage: durable state contracts and implementations
- api: gRPC and future HTTP API definitions
- docs: architecture, decisions, failure model, and roadmap

## Development

Prerequisites:

- Go 1.26 or newer
- Docker with Compose for local MySQL and PostgreSQL integration work

Run:

1. Copy .env.example to .env.
2. Run make check.
3. Run make build.
4. Run make dev-up when Docker is available.

The binaries currently expose their build identity and establish the Phase 0
package boundaries. Functional schema and migration commands are tracked in the
[roadmap](docs/roadmap.md).

## Project status

Development is organized as ten phase milestones. Phase 0 establishes build,
domain, API, storage, local-development, and CI foundations. Phases 1–4 produce
the single-target MVP; Phases 5–8 add fleet-scale orchestration and safety; Phase
9 adds GitOps and packaging.

See [the project-plan review](docs/project-plan-review.md),
[architecture](docs/architecture.md), and [contributing guide](CONTRIBUTING.md).

