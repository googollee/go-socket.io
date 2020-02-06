package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimestamp(t *testing.T) {
	should := assert.New(t)
	t1 := Timestamp()
	t2 := Timestamp()
	should.NotEmpty(t1)
	should.NotEmpty(t2)
	should.NotEqual(t1, t2)
}
