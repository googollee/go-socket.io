// Package packet is codec of packet for connection which supports framing.
package packet

import (
	"io"
)

// FrameReader is the reader which supports framing.
type FrameReader interface {
	NextReader() (FrameType, io.ReadCloser, error)
}

// FrameWriter is the writer which supports framing.
type FrameWriter interface {
	NextWriter(typ FrameType) (io.WriteCloser, error)
}

// NewEncoder creates a packet encoder which writes to w.
func NewEncoder(w FrameWriter) *encoder {
	return newEncoder(w)
}

// NewDecoder creates a packet decoder which reads from r.
func NewDecoder(r FrameReader) *decoder {
	return newDecoder(r)
}
