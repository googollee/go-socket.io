package transport

import (
	"net/http"
	"testing"

	"github.com/googollee/go-engine.io/base"
	"github.com/stretchr/testify/assert"
)

type fakeTransport struct{}

func (f fakeTransport) ServeHTTP(http.Header, http.ResponseWriter, *http.Request) {}

func (f fakeTransport) ConnChan() <-chan base.Conn {
	return nil
}

func TestManager(t *testing.T) {
	at := assert.New(t)
	t1 := fakeTransport{}
	t2 := fakeTransport{}

	m := NewManager()
	m.Register("t1", t1)
	m.Register("t2", t2)

	p := false
	func() {
		defer func() {
			recover()
			p = true
		}()

		m.Register("panic", nil)
	}()
	at.True(p)

	tg := m.Get("t1")
	at.Equal(t1, tg)

	tg = m.Get("nonexist")
	at.Nil(tg)

	names := m.OtherNames("t1")
	at.Equal([]string{"t2"}, names)
}
