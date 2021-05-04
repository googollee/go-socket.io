package base

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testClock struct {
	now time.Time
}

func (c testClock) Now() time.Time {
	return c.now
}

func TestTimestampFromClock(t *testing.T) {
	ts1 := TimestampFromClock(testClock{time.Unix(0, 1000)})
	ts2 := TimestampFromClock(testClock{time.Unix(0, 2000)})

	require.NotEmpty(t, ts1)
	require.NotEmpty(t, ts2)

	assert.NotEqual(t, ts1, ts2)
}
