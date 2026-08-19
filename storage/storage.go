package storage

import (
	"context"
	"time"
)

type TargetID string
type VersionID string
type DeploymentID string

type DeploymentClaim struct {
	ID                 DeploymentID
	TargetID           TargetID
	Version            VersionID
	SourceFingerprint  string
	DesiredFingerprint string
	ClaimedAt          time.Time
}

type AuditEvent struct {
	DeploymentID DeploymentID
	Kind         string
	Payload      []byte
	OccurredAt   time.Time
}

// Store defines durability and atomicity requirements at the domain boundary.
// Implementations must enforce the target/version uniqueness constraint inside
// ClaimDeployment; callers must never use a read-then-insert claim sequence.
type Store interface {
	ClaimDeployment(context.Context, DeploymentClaim) error
	GetDeployment(context.Context, DeploymentID) (DeploymentClaim, error)
	AppendAuditEvent(context.Context, AuditEvent) error
}
