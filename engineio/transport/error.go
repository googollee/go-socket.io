package transport

type httpError struct {
	error
	code int
}

func HTTPErr(err error, code int) HTTPError {
	return &httpError{
		error: err,
		code:  code,
	}
}

func (e httpError) Code() int {
	return e.code
}

func (e httpError) Unwrap() error {
	return e.error
}
