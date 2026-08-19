# Testing and Fault Harness

`make check` is deterministic and does not require Docker. It runs formatting,
the protobuf lint/generation/breaking checks, vet, race-enabled unit tests, and a
build.

The integration gate uses the pinned Phase 0 services:

```sh
make dev-up
GHOSTFLEET_POSTGRES_DSN='postgres://ghostfleet:ghostfleet@127.0.0.1:5432/ghostfleet?sslmode=disable' \
GHOSTFLEET_MYSQL_DSN='ghostfleet:ghostfleet@tcp(127.0.0.1:3306)/ghostfleet?parseTime=true' \
go test -race -tags=integration ./internal/testharness
make dev-down
```

CI starts PostgreSQL 17 and MySQL 8.4 as isolated service containers and runs
the same integration test with the service DSNs. The schema fixture is loaded
before each case and dropped in a deferred cleanup path.

## Failure matrix

| Injection point | Expected state | Retry behavior | Invariant asserted |
| --- | --- | --- | --- |
| Process-crash hook before a state transition | Caller returns an injected fault; no completion is recorded | Restart and re-enter with the same idempotency key | INV-002, INV-005, INV-007 |
| Database-disconnect hook during fixture execution | Current statement fails and later statements do not claim success | Reconnect, reload, and cleanup explicitly | INV-007, INV-010 |
| Failure-artifact collection after either fault | Logs and snapshots are written with secret values redacted | Artifact names and contents are deterministic | INV-005, INV-007 |

The harness models process crashes through an explicit hook so the test process is
not terminated. A production caller must treat that hook as a restart boundary.
Unknown configuration or missing DSNs skip only the opt-in integration suite;
the default unit and CI checks fail explicitly for implementation errors.

