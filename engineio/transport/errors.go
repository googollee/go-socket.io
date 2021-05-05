package transport

import "errors"

// ErrInvalidFrame is returned when writing invalid frame type.
var ErrInvalidFrame = errors.New("invalid frame type")
