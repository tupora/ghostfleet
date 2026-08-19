# Contributing

GhostFleet is safety-critical infrastructure. Changes should be small, reviewable,
and tied to an issue with explicit acceptance criteria.

## Local checks

Run make check to verify formatting, static analysis, tests, and compilation.
Integration tests will use the services in compose.yaml as those suites are
introduced.

## Pull requests

A pull request should:

- explain the user or operator impact;
- identify affected safety invariants;
- include tests for state transitions and failure paths;
- document compatibility or migration implications;
- avoid combining unrelated refactors with behavior changes.

Commits use an imperative, concise subject. Breaking storage, API, schema, or plan
format changes require an architecture decision record.

