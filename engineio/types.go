package engineio

import (
	"net/http"
)

// CheckerFunc is function to check request.
type CheckerFunc func(*http.Request) (http.Header, error)

// ConnInitiatorFunc is function to do after create connection.
type ConnInitiatorFunc func(*http.Request, Conn)
