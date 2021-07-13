package frame

import (
	"io"
)

type Frame struct {
	Type Type
	Data io.Reader
}
