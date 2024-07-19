package cmdErrors

import "fmt"

// PIDNotFound is a custom error type for PID file not found errors.
type PIDNotFound struct {
	Message string
}

// Implement the Error() method from the error interface.
func (e *PIDNotFound) Error() string {
	return e.Message
}

// NewPIDNotFound creates a new PIDNotFound error with the given message.
func NewPIDNotFound() error {
	return &PIDNotFound{Message: "PID file not found"}
}

func (e *PIDNotFound) Is(target error) bool {
	_, ok := target.(*PIDNotFound)
	return ok
}

type WorkSpaceInitError struct {
	Message string
}

func (w *WorkSpaceInitError) Error() string {
	return w.Message
}
func NewWorkSpaceInitError(msg string) *WorkSpaceInitError {
	return &WorkSpaceInitError{Message: msg}
}

func (w *WorkSpaceInitError) Is(target error) bool {
	_, ok := target.(*WorkSpaceInitError)
	return ok
}

var ErrPIDNotFound = NewPIDNotFound()
var ErrWorkSpaceInit = func(msg string, args ...interface{}) *WorkSpaceInitError {
	return NewWorkSpaceInitError(
		fmt.Sprintf(msg, args...),
	)
}
