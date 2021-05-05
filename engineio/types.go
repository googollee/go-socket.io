package engineio

import (
	"net/http"
)

// CheckerFunc is function to check request.
type CheckerFunc func(*http.Request) (http.Header, error)

// ConnInitorFunc is function to do after create connection.
type ConnInitorFunc func(*http.Request, Conn)
