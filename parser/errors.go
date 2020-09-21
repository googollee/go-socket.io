package parser

import "errors"

var (
	ErrInvalidPacketType = errors.New("invalid packet type")

	errInvalidBinaryBufferType = errors.New("buffer packet should be binary")

	errInvalidFirstPacketType = errors.New("first packet should be text frame")

	errFailedBufferAddress = errors.New("can't get Buffer address")
)
