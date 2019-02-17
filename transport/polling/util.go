package polling

import (
	"errors"
	"mime"
	"strings"
)

func mimeSupportBinary(m string) (bool, error) {
	typ, params, err := mime.ParseMediaType(m)
	if err != nil {
		return false, err
	}
	switch typ {
	case "application/octet-stream":
		return true, nil
	case "text/plain":
		charset := strings.ToLower(params["charset"])
		if charset != "utf-8" {
			return false, errors.New("invalid charset")
		}
		return false, nil
	}
	return false, errors.New("invalid content-type")
}

type Addr struct {
	Host string
}

func (a Addr) Network() string {
	return "tcp"
}

func (a Addr) String() string {
	return a.Host
}
