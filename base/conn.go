package base

import (
	"io"
	"net"
	"net/http"
)

// FrameType is the type of frames.
type FrameType byte

const (
	// FrameString identifies a string frame.
	FrameString FrameType = iota
	// FrameBinary identifies a binary frame.
	FrameBinary
	// FrameInvalid identifies a invalid frame.
	FrameInvalid
)

// ByteToFrameType converts a byte to FrameType.
func ByteToFrameType(b byte) FrameType {
	ret := FrameType(b)
	if ret > FrameInvalid {
		return FrameInvalid
	}
	return ret
}

// Byte returns type in byte.
func (t FrameType) Byte() byte {
	return byte(t)
}

// FrameReader reads a frame. It need be closed before next reading.
type FrameReader interface {
	NextReader() (FrameType, PacketType, io.Reader, error)
}

// FrameWriter writes a frame. It need be closed before next writing.
type FrameWriter interface {
	NextWriter(ft FrameType, pt PacketType) (io.WriteCloser, error)
}

// Conn is a connection.
type Conn interface {
	FrameReader
	FrameWriter
	io.Closer
	LocalAddr() net.Addr
	RemoteAddr() net.Addr

	RemoteHeader() http.Header
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
