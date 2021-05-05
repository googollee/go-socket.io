package base

import "io"

// FrameType is the type of frames.
type FrameType byte

// Byte returns type in byte.
func (t FrameType) Byte() byte {
	return byte(t)
}

const (
	// FrameString identifies a string frame.
	FrameString FrameType = iota
	// FrameBinary identifies a binary frame.
	FrameBinary
)

// ByteToFrameType converts a byte to FrameType.
func ByteToFrameType(b byte) FrameType {
	return FrameType(b)
}

// FrameReader reads a frame. It need be closed before next reading.
type FrameReader interface {
	NextReader() (FrameType, PacketType, io.ReadCloser, error)
}

// FrameWriter writes a frame. It need be closed before next writing.
type FrameWriter interface {
	NextWriter(ft FrameType, pt PacketType) (io.WriteCloser, error)
}
