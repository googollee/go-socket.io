package parser

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// Buffer is an binary buffer handler used in emit args. All buffers will be
// sent as binary in the transport layer.
type Buffer struct {
	num      uint64
	isBinary bool

	Data []byte
}

type BufferData struct {
	Num         uint64
	PlaceHolder bool `json:"_placeholder"`
	Data        []byte
}

// MarshalJSON marshals to JSON.
func (a Buffer) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := a.marshalJSONBuf(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (a *Buffer) marshalJSONBuf(buf *bytes.Buffer) error {
	encode := a.encodeText
	if a.isBinary {
		encode = a.encodeBinary
	}

	return encode(buf)
}

func (a *Buffer) encodeText(buf *bytes.Buffer) error {
	buf.WriteString(`{"type":"Buffer","data":[`)
	for i, d := range a.Data {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(strconv.Itoa(int(d)))
	}
	buf.WriteString("]}")

	return nil
}

func (a *Buffer) encodeBinary(buf *bytes.Buffer) error {
	buf.WriteString(`{"_placeholder":true,"num":`)
	buf.WriteString(strconv.FormatUint(a.num, 10))
	buf.WriteString("}")

	return nil
}

// UnmarshalJSON unmarshal data from JSON.
func (a *Buffer) UnmarshalJSON(b []byte) error {
	var data BufferData
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}

	a.isBinary = data.PlaceHolder
	a.Data = data.Data
	a.num = data.Num

	return nil
}
