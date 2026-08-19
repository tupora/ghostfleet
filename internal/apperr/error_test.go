package apperr

import (
	"errors"
	"testing"
)

func TestIsCodeThroughWrappedError(t *testing.T) {
	err := errors.Join(errors.New("operation failed"), &Error{
		Code:    CodeDriftDetected,
		Message: "unexpected index",
	})
	if !IsCode(err, CodeDriftDetected) {
		t.Fatal("IsCode() = false, want true")
	}
	if IsCode(err, CodeNotFound) {
		t.Fatal("IsCode() matched the wrong code")
	}
}
