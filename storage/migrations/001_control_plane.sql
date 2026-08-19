-- GhostFleet control-plane schema, PostgreSQL 17.
-- Apply inside one transaction; the migration runner records the checksum.

CREATE TABLE schema_versions (
    version_id TEXT PRIMARY KEY,
    target_id TEXT NOT NULL,
    parent_version_id TEXT,
    checksum TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX schema_versions_target_version_idx
    ON schema_versions (target_id, version_id);

CREATE TABLE deployments (
    deployment_id TEXT PRIMARY KEY,
    target_id TEXT NOT NULL,
    version_id TEXT NOT NULL,
    source_fingerprint TEXT NOT NULL,
    desired_fingerprint TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('PENDING', 'RUNNING', 'COMPLETED', 'PAUSED')),
    created_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    UNIQUE (target_id, version_id),
    FOREIGN KEY (version_id) REFERENCES schema_versions (version_id)
);

CREATE TABLE tasks (
    task_id TEXT PRIMARY KEY,
    deployment_id TEXT NOT NULL REFERENCES deployments (deployment_id),
    state TEXT NOT NULL CHECK (state IN ('PENDING', 'RUNNING', 'COMPLETED', 'PAUSED')),
    worker_id TEXT,
    generation BIGINT NOT NULL DEFAULT 0 CHECK (generation >= 0),
    attempt INTEGER NOT NULL DEFAULT 0 CHECK (attempt >= 0),
    lease_expires_at TIMESTAMPTZ,
    idempotency_key TEXT,
    UNIQUE (deployment_id)
);

CREATE TABLE leases (
    task_id TEXT NOT NULL REFERENCES tasks (task_id),
    generation BIGINT NOT NULL CHECK (generation > 0),
    worker_id TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    claimed_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (task_id, generation)
);

CREATE TABLE plans (
    deployment_id TEXT PRIMARY KEY REFERENCES deployments (deployment_id),
    format_version INTEGER NOT NULL CHECK (format_version > 0),
    source_fingerprint TEXT NOT NULL,
    destination_fingerprint TEXT NOT NULL,
    plan_bytes BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE target_history (
    target_id TEXT NOT NULL,
    version_id TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (target_id, version_id),
    FOREIGN KEY (version_id) REFERENCES schema_versions (version_id)
);

CREATE TABLE audit_events (
    event_id BIGSERIAL PRIMARY KEY,
    deployment_id TEXT REFERENCES deployments (deployment_id),
    event_kind TEXT NOT NULL,
    payload JSONB NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE OR REPLACE FUNCTION ghostfleet_reject_immutable_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'immutable GhostFleet record: %', TG_TABLE_NAME
        USING ERRCODE = '55006';
END;
$$;

CREATE TRIGGER plans_immutable
    BEFORE UPDATE OR DELETE ON plans
    FOR EACH ROW EXECUTE FUNCTION ghostfleet_reject_immutable_mutation();

CREATE TRIGGER target_history_immutable
    BEFORE UPDATE OR DELETE ON target_history
    FOR EACH ROW EXECUTE FUNCTION ghostfleet_reject_immutable_mutation();

CREATE TRIGGER audit_events_immutable
    BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION ghostfleet_reject_immutable_mutation();

-- Claim SQL must execute in the same transaction as its lease insert:
--
-- UPDATE tasks
-- SET state = 'RUNNING', worker_id = $2, generation = $3,
--     attempt = attempt + 1, lease_expires_at = $4, idempotency_key = $5
-- WHERE task_id = $1
--   AND (lease_expires_at IS NULL OR lease_expires_at <= CURRENT_TIMESTAMP)
--   AND generation < $3
-- RETURNING task_id, deployment_id, generation;
--
-- If no row returns, the caller must distinguish an unexpired lease from a
-- stale generation and return CONFLICT or STALE_FENCE without changing state.
