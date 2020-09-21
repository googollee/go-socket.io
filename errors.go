package socketio

import (
	"errors"
	"fmt"
)

var (
	errUnavailableRootHandler = errors.New("root ('/') doesn't have a namespace handler")

	errFailedConnetNamespace = errors.New("failed connect to namespace without handler")
)

type errorMessage struct {
	namespace string

	err error
}

func (e errorMessage) Error() string {
	return fmt.Sprintf("error in namespace: (%s) with error: (%s)", e.namespace, e.err.Error())
}

func newErrorMessage(namespace string, err error) *errorMessage {
	return &errorMessage{
		namespace: namespace,
		err:       err,
	}
}
