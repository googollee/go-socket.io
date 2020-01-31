package payload

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpError(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		op        string
		err       error
		temporary bool
		errString string
	}{
		{"read", errPaused, true, "read: paused"},
		{"read", errTimeout, false, "read: timeout"},
	}

	for _, test := range tests {
		var err error
		err = newOpError(test.op, test.err)

		assert.Equal(test.errString, err.Error())

		re, ok := err.(Error)
		assert.True(ok)

		assert.Equal(test.temporary, re.Temporary())
	}
}
