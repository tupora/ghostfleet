package storage

import (
	"context"
	"sync"
	"time"

	"github.com/tupora/ghostfleet/internal/apperr"
)

type Version struct {
	ID          VersionID
	TargetID    TargetID
	ParentID    VersionID
	Checksum    string
	Fingerprint string
	CreatedAt   time.Time
}

type DeploymentState string

const (
	DeploymentPending   DeploymentState = "PENDING"
	DeploymentRunning   DeploymentState = "RUNNING"
	DeploymentCompleted DeploymentState = "COMPLETED"
	DeploymentPaused    DeploymentState = "PAUSED"
)

type Deployment struct {
	ID                 DeploymentID
	TargetID           TargetID
	VersionID          VersionID
	SourceFingerprint  string
	DesiredFingerprint string
	State              DeploymentState
	CreatedAt          time.Time
	CompletedAt        time.Time
}

type TaskState string

const (
	TaskPending   TaskState = "PENDING"
	TaskRunning   TaskState = "RUNNING"
	TaskCompleted TaskState = "COMPLETED"
	TaskPaused    TaskState = "PAUSED"
)

type Task struct {
	ID             string
	DeploymentID   DeploymentID
	State          TaskState
	WorkerID       string
	Generation     uint64
	Attempt        uint32
	LeaseExpiresAt time.Time
	IdempotencyKey string
}

type CreateTaskRequest struct {
	ID           string
	DeploymentID DeploymentID
}

type ClaimTaskRequest struct {
	TaskID         string
	WorkerID       string
	Generation     uint64
	LeaseDuration  time.Duration
	IdempotencyKey string
}

type TargetObservation struct {
	TargetID    TargetID
	Known       bool
	VersionID   VersionID
	Fingerprint string
}

type ReconciliationAction string

const (
	ReconciliationContinue ReconciliationAction = "CONTINUE"
	ReconciliationPause    ReconciliationAction = "PAUSE"
)

type Reconciliation struct {
	Action ReconciliationAction
	Reason string
}

// ControlPlane captures the conditional-write and reconciliation contract. A
// PostgreSQL implementation must enforce the same invariants in the migration
// schema; MemoryStore provides deterministic unit and concurrency evidence.
type ControlPlane interface {
	Store
	CreateVersion(context.Context, Version) error
	CreateDeployment(context.Context, Deployment) error
	CreateTask(context.Context, CreateTaskRequest) error
	ClaimTask(context.Context, ClaimTaskRequest) (Task, error)
	CompleteTask(context.Context, string, string, uint64) error
	CompleteDeployment(context.Context, DeploymentID, string, uint64) error
	ReconcileTarget(context.Context, TargetObservation) (Reconciliation, error)
}

// MemoryStore is a concurrency-safe reference implementation of the control-
// plane contract. It is intended for deterministic tests and local wiring, not
// durable production state.
type MemoryStore struct {
	mu          sync.Mutex
	now         func() time.Time
	versions    map[string]Version
	versionIDs  map[VersionID]string
	deployments map[DeploymentID]Deployment
	deploymentK map[string]DeploymentID
	tasks       map[string]Task
	history     map[TargetID]TargetObservation
	audits      []AuditEvent
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		now:         time.Now,
		versions:    make(map[string]Version),
		versionIDs:  make(map[VersionID]string),
		deployments: make(map[DeploymentID]Deployment),
		deploymentK: make(map[string]DeploymentID),
		tasks:       make(map[string]Task),
		history:     make(map[TargetID]TargetObservation),
	}
}

func NewMemoryStoreWithClock(now func() time.Time) *MemoryStore {
	store := NewMemoryStore()
	store.now = now
	return store
}

func (s *MemoryStore) CreateVersion(ctx context.Context, version Version) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := targetVersionKey(version.TargetID, version.ID)
	if existing, ok := s.versions[key]; ok {
		if existing == version {
			return nil
		}
		return conflict("version is immutable")
	}
	if existingKey, ok := s.versionIDs[version.ID]; ok && existingKey != key {
		return conflict("version ID is already used by another target")
	}
	s.versions[key] = version
	s.versionIDs[version.ID] = key
	return nil
}

func (s *MemoryStore) CreateDeployment(ctx context.Context, deployment Deployment) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.deployments[deployment.ID]; ok {
		return conflict("deployment is immutable")
	}
	key := targetVersionKey(deployment.TargetID, deployment.VersionID)
	if existing, ok := s.deploymentK[key]; ok {
		if existing == deployment.ID {
			return nil
		}
		return conflict("target/version deployment already exists")
	}
	if _, ok := s.versions[key]; !ok {
		return notFound("schema version")
	}
	if deployment.State == "" {
		deployment.State = DeploymentPending
	}
	s.deployments[deployment.ID] = deployment
	s.deploymentK[key] = deployment.ID
	return nil
}

func (s *MemoryStore) CreateTask(ctx context.Context, request CreateTaskRequest) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[request.ID]; ok {
		return conflict("task is immutable")
	}
	if _, ok := s.deployments[request.DeploymentID]; !ok {
		return notFound("deployment")
	}
	s.tasks[request.ID] = Task{ID: request.ID, DeploymentID: request.DeploymentID, State: TaskPending}
	return nil
}

func (s *MemoryStore) ClaimTask(ctx context.Context, request ClaimTaskRequest) (Task, error) {
	if err := contextError(ctx); err != nil {
		return Task{}, err
	}
	if request.WorkerID == "" || request.Generation == 0 || request.LeaseDuration <= 0 || request.IdempotencyKey == "" {
		return Task{}, failedPrecondition("worker, generation, lease duration, and idempotency key are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[request.TaskID]
	if !ok {
		return Task{}, notFound("task")
	}
	now := s.now()
	if task.State == TaskCompleted {
		return Task{}, conflict("completed task cannot be claimed")
	}
	if task.WorkerID == request.WorkerID && task.Generation == request.Generation && task.IdempotencyKey == request.IdempotencyKey && task.LeaseExpiresAt.After(now) {
		return task, nil
	}
	if task.LeaseExpiresAt.After(now) {
		return Task{}, conflict("task lease is held by another worker")
	}
	if request.Generation <= task.Generation {
		return Task{}, staleFence("task generation is stale")
	}
	task.State = TaskRunning
	task.WorkerID = request.WorkerID
	task.Generation = request.Generation
	task.Attempt++
	task.LeaseExpiresAt = now.Add(request.LeaseDuration)
	task.IdempotencyKey = request.IdempotencyKey
	s.tasks[task.ID] = task
	return task, nil
}

func (s *MemoryStore) CompleteTask(ctx context.Context, taskID, workerID string, generation uint64) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[taskID]
	if !ok {
		return notFound("task")
	}
	if task.WorkerID != workerID || task.Generation != generation || !task.LeaseExpiresAt.After(s.now()) {
		return staleFence("task owner is no longer current")
	}
	task.State = TaskCompleted
	task.LeaseExpiresAt = time.Time{}
	s.tasks[taskID] = task
	return nil
}

func (s *MemoryStore) CompleteDeployment(ctx context.Context, deploymentID DeploymentID, workerID string, generation uint64) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	deployment, ok := s.deployments[deploymentID]
	if !ok {
		return notFound("deployment")
	}
	for _, task := range s.tasks {
		if task.DeploymentID == deploymentID && (task.WorkerID != workerID || task.Generation != generation || task.State != TaskCompleted) {
			return staleFence("deployment completion is not owned by the current task")
		}
	}
	deployment.State = DeploymentCompleted
	deployment.CompletedAt = s.now()
	s.deployments[deploymentID] = deployment
	return nil
}

func (s *MemoryStore) ReconcileTarget(ctx context.Context, observation TargetObservation) (Reconciliation, error) {
	if err := contextError(ctx); err != nil {
		return Reconciliation{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history[observation.TargetID] = observation
	if !observation.Known {
		return Reconciliation{Action: ReconciliationPause, Reason: "target history is unknown"}, nil
	}
	var latest Deployment
	for _, deployment := range s.deployments {
		if deployment.TargetID == observation.TargetID && deployment.State == DeploymentCompleted && (latest.CompletedAt.IsZero() || deployment.CompletedAt.After(latest.CompletedAt)) {
			latest = deployment
		}
	}
	if latest.ID == "" {
		return Reconciliation{Action: ReconciliationContinue, Reason: "no completed deployment is recorded"}, nil
	}
	if latest.VersionID != observation.VersionID || latest.DesiredFingerprint != observation.Fingerprint {
		return Reconciliation{Action: ReconciliationPause, Reason: "target history disagrees with control-plane history"}, nil
	}
	return Reconciliation{Action: ReconciliationContinue, Reason: "target history matches control-plane history"}, nil
}

func (s *MemoryStore) ClaimDeployment(ctx context.Context, claim DeploymentClaim) error {
	return s.CreateDeployment(ctx, Deployment{
		ID:                 claim.ID,
		TargetID:           claim.TargetID,
		VersionID:          claim.Version,
		SourceFingerprint:  claim.SourceFingerprint,
		DesiredFingerprint: claim.DesiredFingerprint,
		CreatedAt:          claim.ClaimedAt,
	})
}

func (s *MemoryStore) GetDeployment(ctx context.Context, id DeploymentID) (DeploymentClaim, error) {
	if err := contextError(ctx); err != nil {
		return DeploymentClaim{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	deployment, ok := s.deployments[id]
	if !ok {
		return DeploymentClaim{}, notFound("deployment")
	}
	return DeploymentClaim{
		ID:                 deployment.ID,
		TargetID:           deployment.TargetID,
		Version:            deployment.VersionID,
		SourceFingerprint:  deployment.SourceFingerprint,
		DesiredFingerprint: deployment.DesiredFingerprint,
		ClaimedAt:          deployment.CreatedAt,
	}, nil
}

func (s *MemoryStore) AppendAuditEvent(ctx context.Context, event AuditEvent) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	event.Payload = append([]byte(nil), event.Payload...)
	s.audits = append(s.audits, event)
	return nil
}

func (s *MemoryStore) Task(id string) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	return task, ok
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func targetVersionKey(target TargetID, version VersionID) string {
	return string(target) + "\x00" + string(version)
}

func conflict(message string) error {
	return &apperr.Error{Code: apperr.CodeConflict, Message: message}
}

func failedPrecondition(message string) error {
	return &apperr.Error{Code: apperr.CodeFailedPrecond, Message: message}
}

func notFound(resource string) error {
	return &apperr.Error{Code: apperr.CodeNotFound, Message: resource + " not found"}
}

func staleFence(message string) error {
	return &apperr.Error{Code: apperr.CodeStaleFence, Message: message}
}
