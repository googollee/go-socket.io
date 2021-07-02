package polling

import (
	"errors"
)

const (
	separator    = 0x1e
	binaryPrefix = 'b'
)

var (
	ErrNoEnoughBuf          = errors.New("not enough buf to push back")
	ErrNoSpace              = errors.New("no enough space to write")
	ErrPingTimeout          = errors.New("ping timeout")
	ErrSeparatorInTextFrame = errors.New("should not write 0x1e to text frames")
	ErrNonCloseFrame        = errors.New("has a non-closed frame")
)
