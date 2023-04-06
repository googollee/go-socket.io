package packet

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/googollee/go-socket.io/engineio/frame"
)

func TestEncoder(t *testing.T) {
	at := assert.New(t)

	for _, test := range tests {
		w := NewFakeConnWriter()
		encoder := NewEncoder(w)
		for _, p := range test.packets {
			fw, err := encoder.NextWriter(p.FType, p.PType)
			at.Nil(err)
			_, err = fw.Write(p.Data)
			at.Nil(err)
			err = fw.Close()
			at.Nil(err)
		}
		at.Equal(test.frames, w.Frames)
	}
}

func BenchmarkEncoder(b *testing.B) {
	encoder := NewEncoder(&FakeDiscardWriter{})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w, err := encoder.NextWriter(frame.String, MESSAGE)
		if err != nil {
			b.Error(err)
		}

		err = w.Close()
		if err != nil {
			b.Error(err)
		}

		w, err = encoder.NextWriter(frame.Binary, MESSAGE)
		if err != nil {
			b.Error(err)
		}

		err = w.Close()
		if err != nil {
			b.Error(err)
		}
	}
}
