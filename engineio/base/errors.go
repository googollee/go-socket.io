package base

import (
	"fmt"
	"net"
)

// OpError is the error type usually returned by functions in the transport package.
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
