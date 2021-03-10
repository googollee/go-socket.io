package base

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type testClock struct {
	now time.Time
}

func (c testClock) Now() time.Time {
	return c.now
}

func TestTimestampFromClock(t *testing.T) {
	should := assert.New(t)
	t1 := TimestampFromClock(testClock{time.Unix(0, 1000)})
	t2 := TimestampFromClock(testClock{time.Unix(0, 2000)})
	should.NotEmpty(t1)
	should.NotEmpty(t2)
	should.NotEqual(t1, t2)
}
