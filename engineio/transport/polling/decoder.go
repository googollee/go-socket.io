package polling

import (
	"encoding/base64"
	"fmt"
	"io"

	"github.com/googollee/go-socket.io/engineio/frame"
)

// decoder decodes frames from the reader.
type decoder struct {
	reader    *bufReader
	lastFrame *frameReader
}

func newDecoder(buf []byte, r io.Reader) *decoder {
	return &decoder{
		reader: newBufReader(buf, r),
	}
}

// NextFrame returns a reader to read the next frame.
func (d *decoder) NextFrame() (frame.Type, io.Reader, error) {
	if d.lastFrame != nil {
		if err := d.lastFrame.Discard(); err != nil {
			return 0, nil, err
		}
	}

	if err := d.reader.Fill(); err != nil {
		return 0, nil, err
	}

	next, err := d.reader.ReadByte()
	if err != nil && err != io.EOF {
		return 0, nil, err
	}

	if d.lastFrame != nil {
		// Start from the 2nd frame, need to ignore the separator.
		next, err = d.reader.ReadByte()
		if err != nil && err != io.EOF {
			return 0, nil, err
		}
	}

	d.lastFrame = &frameReader{
		reader:   d.reader,
		finished: false,
	}

	if err == nil && next == binaryPrefix {
		return frame.Binary, base64.NewDecoder(base64.StdEncoding, d.lastFrame), nil
	}

	// The next byte is not binary prefix byte. Push it back to the reader
	if err == nil {
		if err := d.lastFrame.reader.PushBack(1); err != nil {
			return 0, nil, err
		}
	}

	return frame.Text, d.lastFrame, nil
}

// frameReader is a reader to read one frame.
type frameReader struct {
	reader   *bufReader
	finished bool
}

// Read reads data of the frame to buffer b.
func (r *frameReader) Read(b []byte) (int, error) {
	n, err := r.reader.Read(b)
	for i := 0; i < n; i++ {
		if b[i] == separator {
			r.finished = true
			if err := r.reader.PushBack(n - i); err != nil {
				return n, fmt.Errorf("decode package error:(it should not happen) %w", err)
			}
			n = i
		}
	}

	if err == io.EOF {
		return n, io.EOF
	}

	if n == 0 && err == nil {
		return 0, io.EOF
	}

	return n, err
}

// ReadByte reads a byte in the frame.
func (r *frameReader) ReadByte() (byte, error) {
	ret, err := r.reader.ReadByte()
	if err != nil {
		return 0, err
	}

	if ret == separator {
		if err := r.reader.PushBack(1); err != nil {
			return 0, fmt.Errorf("decode package error:(it should not happen) %w", err)
		}
		return 0, io.EOF
	}

	return ret, nil
}

// Discard discards all data in the frame.
func (r *frameReader) Discard() error {
	var buf [1024]byte
	for {
		_, err := r.Read(buf[:])
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}

	return nil
}
