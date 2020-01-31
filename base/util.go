package base

import "time"

var chars = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-_")

// Timestamp returns a string based on different nano time.
func Timestamp() string {
	now := time.Now().UnixNano()
	ret := make([]byte, 0, 16)
	for now > 0 {
		ret = append(ret, chars[int(now%int64(len(chars)))])
		now = now / int64(len(chars))
	}
	return string(ret)
}
