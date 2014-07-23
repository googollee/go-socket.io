package socketio

type BroadcastAdaptor interface {
	Join(room string, socket Socket) error
	Leave(room string, socket Socket) error
	Send(ignore Socket, room, message string, args []interface{}) error
}

var newBroadcast = newBroadcastDefault

type broadcast map[string]map[string]Socket

func newBroadcastDefault() BroadcastAdaptor {
	return make(broadcast)
}

func (b broadcast) Join(room string, socket Socket) error {
	sockets, ok := b[room]
	if !ok {
		sockets = make(map[string]Socket)
	}
	sockets[socket.Id()] = socket
	b[room] = sockets
	return nil
}

func (b broadcast) Leave(room string, socket Socket) error {
	sockets, ok := b[room]
	if !ok {
		return nil
	}
	delete(sockets, socket.Id())
	if len(sockets) == 0 {
		delete(b, room)
		return nil
	}
	b[room] = sockets
	return nil
}

func (b broadcast) Send(ignore Socket, room, message string, args []interface{}) error {
	sockets := b[room]
	for id, s := range sockets {
		if ignore != nil && ignore.Id() == id {
			continue
		}
		s.Emit(message, args...)
	}
	return nil
}
