package socketio

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/logger"
	"github.com/googollee/go-socket.io/parser"
)

// Server is a go-socket.io server.
type Client struct {
	conn      *conn
	namespace string
	handlers  *namespaceHandlers
	url       string
	opts      *engineio.Options
}

// NewServer returns a server.
func NewClient(uri string, opts *engineio.Options) (*Client, error) {
	// uri like http://asd.com:8080/namesapce

	url, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	namespace := url.Path

	// Not allowing other than default
	url.Path = path.Join("/socket.io", namespace)
	url.Path = url.EscapedPath()
	if strings.HasSuffix(url.Path, "socket.io") {
		url.Path += "/"
	}

	client := &Client{
		conn:      nil,
		namespace: namespace,
		url:       url.String(),
		handlers:  newNamespaceHandlers(),
		opts:      opts,
	}

	fmt.Println(client)

	return client, nil
}

func (s *Client) Connect() error {
	dialer := engineio.Dialer{
		Transports: []transport.Transport{polling.Default},
	}
	enginioCon, err := dialer.Dial(s.url, nil)
	if err != nil {
		return err
	}

	// Set the engine connection
	c := newConn(enginioCon, s.handlers)

	s.conn = c

	if err := c.connectClient(); err != nil {
		_ = c.Close()
		if root, ok := s.handlers.Get(rootNamespace); ok && root.onError != nil {
			root.onError(nil, err)
		}

		return err
	}

	go s.clientError(c)
	go s.clientWrite(c)
	go s.clientRead(c)
	return nil
}

// Close closes server.
func (s *Client) Close() error {
	return s.conn.Close()
}

func (s *Client) Emit(event string, args ...interface{}) {
	nsp := s.namespace
	if nsp == aliasRootNamespace {
		nsp = rootNamespace
	}

	ns, ok := s.conn.namespaces.Get(nsp)
	if !ok {
		logger.Info("Connection Namespace not initialized")
		return
	}
	ns.Emit(event, args...)
}

// OnConnect set a handler function f to handle open event for namespace.
func (s *Client) OnConnect(f func(Conn) error) {
	h := s.getNamespace(s.namespace)
	if h == nil {
		h = s.createNamespace(s.namespace)
	}

	h.OnConnect(f)
}

// OnDisconnect set a handler function f to handle disconnect event for namespace.
func (s *Client) OnDisconnect(f func(Conn, string)) {
	h := s.getNamespace(s.namespace)
	if h == nil {
		h = s.createNamespace(s.namespace)
	}

	h.OnDisconnect(f)
}

// OnError set a handler function f to handle error for namespace.
func (s *Client) OnError(f func(Conn, error)) {
	h := s.getNamespace(s.namespace)
	if h == nil {
		h = s.createNamespace(s.namespace)
	}

	h.OnError(f)
}

// OnEvent set a handler function f to handle event for namespace.
func (s *Client) OnEvent(event string, f interface{}) {
	h := s.getNamespace(s.namespace)
	if h == nil {
		h = s.createNamespace(s.namespace)
	}

	h.OnEvent(event, f)
}

/////////////////////////
// Private Functions
/////////////////////////

func (s *Client) clientError(c *conn) {
	defer func() {
		if err := c.Close(); err != nil {
			logger.Error("close connect:", err)
		}

	}()

	for {
		select {
		case <-c.quitChan:
			return
		case err := <-c.errorChan:
			logger.Error("clientError", err)

			var errMsg *errorMessage
			if !errors.As(err, &errMsg) {
				continue
			}

			if handler := c.namespace(errMsg.namespace); handler != nil {
				if handler.onError != nil {
					nsConn, ok := c.namespaces.Get(errMsg.namespace)
					if !ok {
						continue
					}
					handler.onError(nsConn, errMsg.err)
				}
			}
		}
	}
}

func (s *Client) clientWrite(c *conn) {
	defer func() {
		if err := c.Close(); err != nil {
			logger.Error("close connect:", err)
		}

	}()

	for {
		select {
		case <-c.quitChan:
			logger.Info("clientWrite Writer loop has stopped")
			return
		case pkg := <-c.writeChan:
			if err := c.encoder.Encode(pkg.Header, pkg.Data); err != nil {
				c.onError(pkg.Header.Namespace, err)
			}
		}
	}
}

func (s *Client) clientRead(c *conn) {
	defer func() {
		if err := c.Close(); err != nil {
			logger.Error("close connect:", err)
		}
	}()

	var event string

	for {
		var header parser.Header

		if err := c.decoder.DecodeHeader(&header, &event); err != nil {
			c.onError(rootNamespace, err)
			logger.Error("clientRead Error in Decoder", err)
			return
		}

		if header.Namespace == aliasRootNamespace {
			header.Namespace = rootNamespace
		}

		var err error
		switch header.Type {
		case parser.Ack:
			err = ackPacketHandler(c, header)
		case parser.Connect:
			err = clientConnectPacketHandler(c, header)
		case parser.Disconnect:
			err = clientDisconnectPacketHandler(c, header)
		case parser.Event:
			err = eventPacketHandler(c, event, header)
		}

		if err != nil {
			logger.Error("client read:", err)
			return
		}
	}
}

func (s *Client) createNamespace(nsp string) *namespaceHandler {
	if nsp == aliasRootNamespace {
		nsp = rootNamespace
	}

	handler := newNamespaceHandler(nsp, nil)
	s.handlers.Set(nsp, handler)

	return handler
}

func (s *Client) getNamespace(nsp string) *namespaceHandler {
	if nsp == aliasRootNamespace {
		nsp = rootNamespace
	}

	ret, ok := s.handlers.Get(nsp)
	if !ok {
		return nil
	}

	return ret
}

////
// Handlers
////

func (c *conn) connectClient() error {
	rootHandler, ok := c.handlers.Get(rootNamespace)
	if !ok {
		return errUnavailableRootHandler
	}

	root := newNamespaceConn(c, aliasRootNamespace, rootHandler.broadcast)
	c.namespaces.Set(rootNamespace, root)

	root.Join(root.Conn.ID())

	c.namespaces.Range(func(ns string, nc *namespaceConn) {
		nc.SetContext(c.Conn.Context())
	})

	header := parser.Header{
		Type: parser.Connect,
	}

	if err := c.encoder.Encode(header); err != nil {
		return err
	}

	return nil
}
