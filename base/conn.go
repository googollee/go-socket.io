package base

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

// OpError is the error type usually returned by functions in the transport
// package.
type OpError struct {
	URL string
	Op  string
	Err error
}

// OpErr makes an *OpError
func OpErr(url, op string, err error) error {
	return &OpError{
		URL: url,
		Op:  op,
		Err: err,
	}
}

func (e *OpError) Error() string {
	return fmt.Sprintf("%s %s: %s", e.Op, e.URL, e.Err.Error())
}

// Timeout returns true if the error is a timeout.
func (e *OpError) Timeout() bool {
	if r, ok := e.Err.(net.Error); ok {
		return r.Timeout()
	}
	return false
}

// Temporary returns true if the error is temporary.
func (e *OpError) Temporary() bool {
	if r, ok := e.Err.(net.Error); ok {
		return r.Temporary()
	}
	return false
}

// FrameType is the type of frames.
type FrameType byte

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

// Byte returns type in byte.
func (t FrameType) Byte() byte {
	return byte(t)
}

// FrameReader reads a frame. It need be closed before next reading.
type FrameReader interface {
	NextReader() (FrameType, PacketType, io.ReadCloser, error)
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
	URL() url.URL
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	RemoteHeader() http.Header
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// ConnParameters is connection parameter of server.
type ConnParameters struct {
	PingInterval time.Duration
	PingTimeout  time.Duration
	SID          string
	Upgrades     []string
}

type jsonParameters struct {
	SID          string   `json:"sid"`
	Upgrades     []string `json:"upgrades"`
	PingInterval int      `json:"pingInterval"`
	PingTimeout  int      `json:"pingTimeout"`
}

// ReadConnParameters reads ConnParameters from r.
func ReadConnParameters(r io.Reader) (ConnParameters, error) {
	var param jsonParameters
	if err := json.NewDecoder(r).Decode(&param); err != nil {
		return ConnParameters{}, err
	}
	return ConnParameters{
		SID:          param.SID,
		Upgrades:     param.Upgrades,
		PingInterval: time.Duration(param.PingInterval) * time.Millisecond,
		PingTimeout:  time.Duration(param.PingTimeout) * time.Millisecond,
	}, nil
}

// WriteTo writes to w with json format.
func (p ConnParameters) WriteTo(w io.Writer) (int64, error) {
	arg := jsonParameters{
		SID:          p.SID,
		Upgrades:     p.Upgrades,
		PingInterval: int(p.PingInterval / time.Millisecond),
		PingTimeout:  int(p.PingTimeout / time.Millisecond),
	}
	writer := writer{
		w: w,
	}
	err := json.NewEncoder(&writer).Encode(arg)
	return writer.i, err
}

type writer struct {
	i int64
	w io.Writer
}

func (w *writer) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.i += int64(n)
	return n, err
}
