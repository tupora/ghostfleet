package engine

import "context"

type Migration struct {
	ID                     string
	TargetID               string
	SourceFingerprint      string
	DestinationFingerprint string
	Statement              string
}

type Status struct {
	Phase    string
	Progress float64
	Message  string
}

type Engine interface {
	Validate(context.Context, Migration) error
	Prepare(context.Context, Migration) error
	Start(context.Context, Migration) error
	Pause(context.Context, Migration) error
	Resume(context.Context, Migration) error
	Status(context.Context, Migration) (Status, error)
	Cutover(context.Context, Migration) error
	Abort(context.Context, Migration) error
}
