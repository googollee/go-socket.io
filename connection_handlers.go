package socketio

import (
	"log"

	"github.com/googollee/go-socket.io/logger"
	"github.com/googollee/go-socket.io/parser"
)

var readHandlerMapping = map[parser.Type]readHandler{
	parser.Ack:        ackPacketHandler,
	parser.Connect:    connectPacketHandler,
	parser.Disconnect: disconnectPacketHandler,
}

func ackPacketHandler(c *conn, header parser.Header) error {
	conn, ok := c.namespaces.Get(header.Namespace)
	if !ok {
		_ = c.decoder.DiscardLast()
		return nil
	}

	conn.dispatch(header)

	return nil
}

func eventPacketHandler(c *conn, event string, header parser.Header) error {
	conn, ok := c.namespaces.Get(header.Namespace)
	if !ok {
		_ = c.decoder.DiscardLast()
		return nil
	}

	handler, ok := c.handlers.Get(header.Namespace)
	if !ok {
		_ = c.decoder.DiscardLast()
		logger.Info("missing handler for namespace", "namespace", header.Namespace)
		return nil
	}

	args, err := c.decoder.DecodeArgs(handler.getEventTypes(event))
	if err != nil {
		c.onError(header.Namespace, err)
		logger.Info("Error decoding the message type", "namespace", header.Namespace, "event", event, "eventType", handler.getEventTypes(event), "err", err.Error())
		return errDecodeArgs
	}

	ret, err := handler.dispatchEvent(conn, event, args...)
	if err != nil {
		c.onError(header.Namespace, err)
		logger.Info("Error for event type", "namespace", header.Namespace, "event", event)
		return errHandleDispatch
	}

	if len(ret) > 0 {
		header.Type = parser.Ack
		c.write(header, ret...)
	}

	return nil
}

func connectPacketHandler(c *conn, header parser.Header) error {
	if err := c.decoder.DiscardLast(); err != nil {
		c.onError(header.Namespace, err)
		logger.Info("connectPacketHandler DiscardLast", err, "namespace", header.Namespace)
		return nil
	}

	handler, ok := c.handlers.Get(header.Namespace)
	if !ok {
		c.onError(header.Namespace, errFailedConnectNamespace)
		logger.Info("connectPacketHandler get namespace handler", "namespace", header.Namespace)
		return errFailedConnectNamespace
	}

	conn, ok := c.namespaces.Get(header.Namespace)
	if !ok {
		conn = newNamespaceConn(c, header.Namespace, handler.broadcast)
		c.namespaces.Set(header.Namespace, conn)
		conn.Join(c.Conn.ID())
	}

	_, err := handler.dispatch(conn, header)
	if err != nil {
		logger.Info("connectPacketHandler dispatch error", "namespace", header.Namespace)
		log.Println("dispatch connect packet", err)
		c.onError(header.Namespace, err)
		return errHandleDispatch
	}

	c.write(header)

	return nil
}

func disconnectPacketHandler(c *conn, header parser.Header) error {
	args, err := c.decoder.DecodeArgs(defaultHeaderType)
	if err != nil {
		c.onError(header.Namespace, err)
		return errDecodeArgs
	}

	conn, ok := c.namespaces.Get(header.Namespace)
	if !ok {
		_ = c.decoder.DiscardLast()
		return nil
	}

	conn.LeaveAll()

	c.namespaces.Delete(header.Namespace)

	handler, ok := c.handlers.Get(header.Namespace)
	if !ok {
		return nil
	}

	_, err = handler.dispatch(conn, header, args...)
	if err != nil {
		log.Println("dispatch disconnect packet", err)
		c.onError(header.Namespace, err)
		return errHandleDispatch
	}

	return nil
}
