package payload

import (
	"errors"
	"fmt"
)

// Error is payload error.
type Error interface {
	Error() string
	Temporary() bool
}

// OpError is operation error.
type OpError struct {
	Op  string
	Err error
}

func newOpError(op string, err error) error {
	return &OpError{
		Op:  op,
		Err: err,
	}
}

func (e *OpError) Error() string {
	return fmt.Sprintf("%s: %s", e.Op, e.Err.Error())
}

// Temporary returns true if error can retry.
func (e *OpError) Temporary() bool {
	if oe, ok := e.Err.(Error); ok {
		return oe.Temporary()
	}
	return false
}

type retryError struct {
	err string
}

func (e retryError) Error() string {
	return e.err
}

func (e retryError) Temporary() bool {
	return true
}

var errPaused = retryError{"paused"}

var errTimeout = errors.New("timeout")

var errInvalidPayload = errors.New("invalid payload")

var errDrain = errors.New("drain")

var errOverlap = errors.New("overlap")
