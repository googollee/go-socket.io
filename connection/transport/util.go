package transport

import (
	"errors"
	"net/http"
)

// Checker is function to check request.
type Checker func(*http.Request) (http.Header, error)

// ErrInvalidFrame is returned when writing invalid frame type.
var ErrInvalidFrame = errors.New("invalid frame type")

// ErrInvalidPacket is returned when writing invalid packet type.
var ErrInvalidPacket = errors.New("invalid packet type")
