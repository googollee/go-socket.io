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
)

const (
	ProtocolVersion = 1
)

type Client struct {
	*EventEmitter
}

func NewClient() *Client {
	return &Client{
		EventEmitter: NewEventEmitter(),
	}
}

func (c *Client) Run(url_, origin string) error {
	u, err := url.Parse(url_)
	if err != nil {
		return err
	}
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
	u.Path = ""

	url_ = fmt.Sprintf("%s/socket.io/%d/", u.String(), ProtocolVersion)
	r, err := http.Get(url_)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		fmt.Println(url_)
		return errors.New("invalid status: " + r.Status)
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	parts := strings.SplitN(string(body), ":", 4)
	if len(parts) != 4 {
		return errors.New("invalid handshake: " + string(body))
	}
	if !strings.Contains(parts[3], "websocket") {
		return errors.New("server does not support websockets")
	}
	sessionId := parts[0]
	wsurl := "ws" + url_[4:]
	wsurl = fmt.Sprintf("%swebsocket/%s", wsurl, sessionId)
	ws, err := websocket.Dial(wsurl, "", origin)
	if err != nil {
		fmt.Println(wsurl)
		return err
	}

	timeout, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return err
	}

	session := NewSession(map[string]*EventEmitter{endpoint: c.EventEmitter}, sessionId, int(timeout), false)
	transport := newWebSocket(session)
	transport.conn = ws
	session.transport = transport
	fmt.Println(endpoint)
	if endpoint != "" {
		session.Of(endpoint).sendPacket(new(connectPacket))
	}
	session.loop()
	return nil
}
