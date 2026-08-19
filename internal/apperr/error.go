package apperr

import (
	"errors"
	"fmt"
)

type Code string

const (
	CodeConflict       Code = "CONFLICT"
	CodeDriftDetected  Code = "DRIFT_DETECTED"
	CodeFailedPrecond  Code = "FAILED_PRECONDITION"
	CodeInternal       Code = "INTERNAL"
	CodeNotFound       Code = "NOT_FOUND"
	CodeStaleFence     Code = "STALE_FENCE"
	CodeVersionApplied Code = "VERSION_ALREADY_APPLIED"
)

type Error struct {
	Code    Code
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Message == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Cause }

func IsCode(err error, code Code) bool {
	var appError *Error
	return errors.As(err, &appError) && appError.Code == code
}
