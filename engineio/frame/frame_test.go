package frame

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrameType(t *testing.T) {
	at := assert.New(t)
	tests := []struct {
		b    byte
		typ  Type
		outb byte
	}{
		{0, String, 0},
		{1, Binary, 1},
	}

	for _, test := range tests {
		typ := ByteToFrameType(test.b)
		at.Equal(test.typ, typ)
		b := typ.Byte()
		at.Equal(test.outb, b)
	}
}
