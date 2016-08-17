// Package payload is framing layer for connection which doesn't support framing.
package payload

import (
	"errors"
	"io"

	"github.com/googollee/go-engine.io/base"
)

// ConnReader is a reader of one connection. It should have buffer internal,
// which can read byte by byte.
type ConnReader interface {
	SupportBinary() bool
	io.Reader
	ReadByte() (byte, error)
}

// ConnWriter is a writer of one connection. WriteFrame called means writing
// one frame into that connection.
type ConnWriter interface {
	SupportBinary() bool
	WriteFrame(head, data []byte) error
}

// Encoder encodes packet frames into a payload. It need be closed before
// sending payload data.
// It can changing output Writer w while using. The senario is, when using xhr
// as connection, it need change BufWriter as output between GET response. It
// must close frame and Flushed before SetWriter.
type Encoder interface {
	base.FrameWriter
}

// NewEncoder creates a new encoder, output to w. The maximum size of one frame
// is limited with maxFrameSize. If w supports binary, set supportBinary true,
// otherwise set it to false.
func NewEncoder(w ConnWriter) Encoder {
	return newEncoder(w)
}

// ErrInvalidPayload is error of invalid payload while decoding.
var ErrInvalidPayload = errors.New("invalid payload")

// Decoder decodes packet from a payload.
// It can be changed input BufReader r while using. The senario is, when using
// xhr as connection, it need change request body as input between POST request.
// It must close frame before SetReader.
type Decoder interface {
	base.FrameReader
}

// NewDecoder creates a new decoder, input from r. If r supports binary, set
// supportBinary true, otherwise set it to false.
func NewDecoder(r ConnReader) Decoder {
	return newDecoder(r)
}
