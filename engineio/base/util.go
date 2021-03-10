package base

import "time"

var chars = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_")

type clock interface {
	Now() time.Time
}

type timeClock struct{}

func (timeClock) Now() time.Time {
	return time.Now()
}

// Timestamp returns a string based on different nano time.
func Timestamp() string {
	return TimestampFromClock(timeClock{})
}

func TimestampFromClock(c clock) string {
	now := c.Now().UnixNano()
	ret := make([]byte, 0, 16)
	for now > 0 {
		ret = append(ret, chars[int(now%int64(len(chars)))])
		now = now / int64(len(chars))
	}
	return string(ret)
}
