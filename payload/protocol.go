// Package payload is framing layer for connection which doesn't support framing.
package payload

import (
	"errors"
	"io"
	"sync/atomic"

	"github.com/googollee/go-engine.io/base"
)

type ByteReader interface {
	ReadByte() (byte, error)
	io.Reader
}

type ByteWriter interface {
	WriteByte(b byte) error
	io.Writer
}

// Encoder encodes packet frames into a payload. It need be closed before
// sending payload data.
// It can changing output Writer w while using. The senario is, when using xhr
// as connection, it need change BufWriter as output between GET response. It
// must close frame and Flushed before SetWriter.
type Encoder interface {
	base.FrameWriter
	FlushOut(w io.Writer) error
}

// NewEncoder creates a new encoder, output to w. The maximum size of one frame
// is limited with maxFrameSize. If w supports binary, set supportBinary true,
// otherwise set it to false.
func NewEncoder(supportBinary bool, closed chan struct{}, err *atomic.Value) Encoder {
	return newEncoder(supportBinary, closed, err)
}

// ErrInvalidPayload is error of invalid payload while decoding.
var ErrInvalidPayload = errors.New("invalid payload")

// Decoder decodes packet from a payload.
// It can be changed input BufReader r while using. The senario is, when using
// xhr as connection, it need change request body as input between POST request.
// It must close frame before SetReader.
type Decoder interface {
	base.FrameReader
	FeedIn(typ base.FrameType, r io.Reader) error
}

// NewDecoder creates a new decoder, input from r. If r supports binary, set
// supportBinary true, otherwise set it to false.
func NewDecoder(closed chan struct{}, err *atomic.Value) Decoder {
	return newDecoder(closed, err)
}
