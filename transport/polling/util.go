package polling

import (
	"errors"
	"io"
	"mime"
	"strings"

	"github.com/googollee/go-engine.io/base"
)

func retError(url, op string, err error) error {
	if err == nil || err == io.EOF {
		return err
	}
	if opErr, ok := err.(*base.OpError); ok {
		return opErr
	}
	return base.OpErr(url, op, err)
}

func normalizeMime(m string) (base.FrameType, error) {
	typ, params, err := mime.ParseMediaType(m)
	if err != nil {
		return 0, err
	}
	switch typ {
	case "application/octet-stream":
		return base.FrameBinary, nil
	case "text/plain":
		charset := strings.ToLower(params["charset"])
		if charset != "utf-8" {
			return 0, errors.New("invalid charset")
		}
		return base.FrameString, nil
	}
	return 0, errors.New("invalid content-type")
}
