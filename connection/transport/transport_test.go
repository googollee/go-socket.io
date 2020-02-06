package transport

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/googollee/go-socket.io/connection/base"

	"github.com/stretchr/testify/assert"
)

type fakeTransport struct {
	name string
}

func (f fakeTransport) Name() string {
	return f.name
}

func (f fakeTransport) Dial(url *url.URL, header http.Header) (base.Conn, error) {
	return nil, nil
}

func (f fakeTransport) Accept(http.ResponseWriter, *http.Request) (base.Conn, error) {
	return nil, nil
}

func TestManager(t *testing.T) {
	at := assert.New(t)
	t1 := fakeTransport{"t1"}
	t2 := fakeTransport{"t2"}
	t3 := fakeTransport{"t3"}
	t4 := fakeTransport{"t4"}

	m := NewManager([]Transport{t1, t2, t3, t4})

	tg := m.Get("t1")
	at.Equal(t1, tg)

	tg = m.Get("nonexist")
	at.Nil(tg)

	names := m.UpgradeFrom("t2")
	at.Equal([]string{"t3", "t4"}, names)

	names = m.UpgradeFrom("t4")
	at.Equal([]string{}, names)

	names = m.UpgradeFrom("nonexist")
	at.Nil(names)
}
