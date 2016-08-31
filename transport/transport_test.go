package transport

import (
	"net/http"
	"testing"

	"github.com/googollee/go-engine.io/base"
	"github.com/stretchr/testify/assert"
)

type fakeTransport struct {
	name string
}

func (f fakeTransport) Name() string {
	return f.name
}

func (f fakeTransport) Dial(url string, header http.Header) (base.Conn, error) {
	return nil, nil
}

func (f fakeTransport) ServeHTTP(chan<- base.Conn, http.ResponseWriter, *http.Request) {}

func TestManager(t *testing.T) {
	at := assert.New(t)
	t1 := fakeTransport{"t1"}
	t2 := fakeTransport{"t2"}

	m := NewManager()
	m.Register(t1)
	m.Register(t2)

	p := false
	func() {
		defer func() {
			recover()
			p = true
		}()

		m.Register(nil)
	}()
	at.True(p)

	tg := m.Get("t1")
	at.Equal(t1, tg)

	tg = m.Get("nonexist")
	at.Nil(tg)

	names := m.OtherNames("t1")
	at.Equal([]string{"t2"}, names)
}
