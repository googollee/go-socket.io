package utils

// ConstError generates a const error.
// ```
// const SomeError = utils.ConstError("some error")
// ```
type ConstError string

// Error returns the message of the error.
func (e ConstError) Error() string {
	return string(e)
}
