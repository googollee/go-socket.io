package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var attachmentTests = []struct {
	buffer         Buffer
	textEncoding   string
	binaryEncoding string
}{
	{
		Buffer{0, false, []byte{1, 255}},
		`{"type":"Buffer","data":[1,255]}`,
		`{"_placeholder":true,"num":0}`,
	},
	{
		Buffer{1, false, []byte{}},
		`{"type":"Buffer","data":[]}`,
		`{"_placeholder":true,"num":1}`,
	},
	{
		Buffer{2, false, nil},
		`{"type":"Buffer","data":[]}`,
		`{"_placeholder":true,"num":2}`,
	},
}

func TestAttachmentEncodeText(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	for _, test := range attachmentTests {
		a := test.buffer
		a.isBinary = false
		j, err := json.Marshal(a)

		must.NoError(err)

		should.Equal(test.textEncoding, string(j))
	}
}

func TestAttachmentEncodeBinary(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	for _, test := range attachmentTests {
		a := test.buffer
		a.isBinary = true
		j, err := json.Marshal(a)

		must.NoError(err)

		should.Equal(test.binaryEncoding, string(j))
	}
}

func TestAttachmentDecodeText(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	for _, test := range attachmentTests {
		var a Buffer
		err := json.Unmarshal([]byte(test.textEncoding), &a)

		must.NoError(err)
		should.False(a.isBinary)

		if len(test.buffer.Data) == 0 {
			should.Equal([]byte{}, a.Data)
			continue
		}

		should.Equal(test.buffer.Data, a.Data)
	}
}

func TestAttachmentDecodeBinary(t *testing.T) {
	should := assert.New(t)
	must := require.New(t)

	for _, test := range attachmentTests {
		var a Buffer
		err := json.Unmarshal([]byte(test.binaryEncoding), &a)

		must.NoError(err)

		should.True(a.isBinary)
		should.Equal(test.buffer.num, a.num)
	}
}
