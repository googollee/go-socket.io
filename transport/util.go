package transport

import (
	"errors"
	"net/http"
)

// HTTPError is error which has http response code
type HTTPError interface {
	Code() int
}

// Checker is function to check request.
type Checker func(*http.Request) (http.Header, error)

// ErrInvalidFrame is returned when writing invalid frame type.
var ErrInvalidFrame = errors.New("invalid frame type")

// ErrInvalidPacket is returned when writing invalid packet type.
var ErrInvalidPacket = errors.New("invalid packet type")
