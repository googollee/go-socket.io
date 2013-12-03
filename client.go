package socketio

import (
	"code.google.com/p/go.net/websocket"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	ProtocolVersion = 1
)

type Client struct {
	session  *Session
	endpoint string
	*EventEmitter
}

func Dial(origin string) (*Client, error) {
	u, err := url.Parse(origin)
	if err != nil {
		return nil, err
	}
	endpoint := parseEndpoint(u)
	u.Path = ""

	url_ := fmt.Sprintf("%s/socket.io/%d/", u.String(), ProtocolVersion)
	r, err := http.Get(url_)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, errors.New("invalid status: " + r.Status)
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(string(body), ":", 4)
	if len(parts) != 4 {
		return nil, errors.New("invalid handshake: " + string(body))
	}
	if !strings.Contains(parts[3], "websocket") {
		return nil, errors.New("server does not support websockets")
	}
	sessionId := parts[0]
	wsurl := "ws" + url_[4:]
	wsurl = fmt.Sprintf("%swebsocket/%s", wsurl, sessionId)
	ws, err := websocket.Dial(wsurl, "", url_)
	if err != nil {
		return nil, err
	}

	timeout, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, err
	}

	ee := NewEventEmitter()
	emitters := make(map[string]*EventEmitter)
	emitters[endpoint] = ee
	if endpoint != "" {
		emitters[""] = NewEventEmitter()
	}
	session := NewSession(emitters, sessionId, int(timeout), false)
	transport := newWebSocket(session)
	transport.conn = ws
	session.transport = transport
	if endpoint != "" {
		session.transport.Send(encodePacket(endpoint, new(connectPacket)))
	}

	return &Client{
		session:      session,
		endpoint:     endpoint,
		EventEmitter: ee,
	}, nil
}

func parseEndpoint(u *url.URL) string {
	path := u.Path
	if l := len(path); l > 0 && path[len(path)-1] == '/' {
		path = path[:l-1]
	}
	lastPath := strings.LastIndex(path, "/")
	endpoint := ""
	if lastPath >= 0 {
		path := path[lastPath:]
		if len(path) > 0 {
			endpoint = path
		}
	}
	return endpoint
}

func (c *Client) Run() {
	c.session.loop()
}

func (c *Client) Quit() error {
	return nil
}

func (c *Client) Of(name string) (nameSpace *NameSpace) {
	ee := c.session.emitters[name]
	ns := c.session.Of(name)
	if ee == nil {
		c.session.transport.Send(encodePacket(name, new(connectPacket)))
		ns.connected = true
	}
	return ns
}

func (c *Client) Call(name string, timeout time.Duration, reply []interface{}, args ...interface{}) error {
	return c.Of(c.endpoint).Call(name, timeout, reply, args...)
}

func (c *Client) Emit(name string, args ...interface{}) error {
	return c.Of(c.endpoint).Emit(name, args...)
}
