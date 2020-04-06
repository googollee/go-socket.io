package socketio

import "sync"

type EachFunc func(Conn)

// Broadcast is the adaptor to handle broadcasts & rooms for socket.io server API
type Broadcast interface {
	Join(room string, connection Conn)                        // Join causes the connection to join a room
	Leave(room string, connection Conn)                       // Leave causes the connection to leave a room
	LeaveAll(connection Conn)                                 // LeaveAll causes given connection to leave all rooms
	Clear(room string)                                        // Clear causes removal of all connections from the room
	Emit(connectID string, event string, args ...interface{}) // Emit will send an event with args to target socket connection
	SendToRoom(room, event string, args ...interface{})       // Send will send an event with args to the room
	SendAll(event string, args ...interface{})                // SendAll will send an event with args to all the rooms
	ForEach(room string, f EachFunc)                          // ForEach will
	Len(room string) int                                      // Len gives number of connections in the room
	Rooms(connection Conn) []string                           // Gives list of all the rooms if no connection given, else list of all the rooms the connection joined
}

// broadcast gives Join, Leave & BroadcastTO server API support to socket.io along with room management
type broadcast struct {
	lock sync.RWMutex // access lock for rooms

	rooms       map[string]map[string]Conn // map of rooms where each room contains a map of connection id to connections in that room
	connections map[string]Conn            // map of connection id to all connections
}

// NewBroadcast creates a new broadcast adapter
func NewBroadcast() Broadcast {
	return &broadcast{
		rooms:       make(map[string]map[string]Conn),
		connections: make(map[string]Conn),
	}
}

// Join joins the given connection to the broadcast room
func (broadcast *broadcast) Join(room string, connection Conn) {
	broadcast.lock.Lock()
	defer broadcast.lock.Unlock()

	if _, ok := broadcast.rooms[room]; !ok {
		broadcast.rooms[room] = make(map[string]Conn)
	}

	broadcast.connections[connection.ID()] = connection
	broadcast.rooms[room][connection.ID()] = connection
}

// Leave leaves the given connection from given room (if exist)
func (broadcast *broadcast) Leave(room string, connection Conn) {
	broadcast.lock.Lock()
	defer broadcast.lock.Unlock()

	if connections, ok := broadcast.rooms[room]; ok {
		delete(connections, connection.ID())

		delete(broadcast.connections, connection.ID())

		if len(connections) == 0 {
			delete(broadcast.rooms, room)
		}
	}
}

// LeaveAll leaves the given connection from all rooms
func (broadcast *broadcast) LeaveAll(connection Conn) {
	broadcast.lock.Lock()
	defer broadcast.lock.Unlock()

	for room, connections := range broadcast.rooms {
		delete(connections, connection.ID())
		delete(broadcast.connections, connection.ID())

		if len(connections) == 0 {
			delete(broadcast.rooms, room)
		}
	}
}

// Clear clears the room
func (broadcast *broadcast) Clear(room string) {
	broadcast.lock.Lock()
	defer broadcast.lock.Unlock()

	for _, connection := range broadcast.rooms[room] {
		delete(broadcast.connections, connection.ID())
	}

	delete(broadcast.rooms, room)
}

// Send sends given event & args to all the connections in the specified room
func (broadcast *broadcast) SendToRoom(room, event string, args ...interface{}) {
	broadcast.lock.RLock()
	defer broadcast.lock.RUnlock()

	for _, connection := range broadcast.rooms[room] {
		connection.Emit(event, args...)
	}
}

// SendAll sends given event & args to all the connections to all the rooms
func (broadcast *broadcast) SendAll(event string, args ...interface{}) {
	broadcast.lock.RLock()
	defer broadcast.lock.RUnlock()

	for _, connections := range broadcast.rooms {
		for _, connection := range connections {
			connection.Emit(event, args...)
		}
	}
}

// SendForEach sends data returned by DataFunc, if the return is 'ok' (second return)
func (broadcast *broadcast) ForEach(room string, f EachFunc) {
	broadcast.lock.RLock()
	defer broadcast.lock.RUnlock()

	occupants, ok := broadcast.rooms[room]
	if !ok {
		return
	}

	for _, connection := range occupants {
		f(connection)
	}
}

// Len gives number of connections in the room
func (broadcast *broadcast) Len(room string) int {
	broadcast.lock.RLock()
	defer broadcast.lock.RUnlock()

	return len(broadcast.rooms[room])
}

// Rooms gives the list of all the rooms available for broadcast in case of
// no connection is given, in case of a connection is given, it gives
// list of all the rooms the connection is joined to
func (broadcast *broadcast) Rooms(connection Conn) []string {
	broadcast.lock.RLock()
	defer broadcast.lock.RUnlock()
	var rooms []string

	if connection == nil {
		rooms = make([]string, 0, len(broadcast.rooms))
		for room := range broadcast.rooms {
			rooms = append(rooms, room)
		}
	} else {
		for room, connections := range broadcast.rooms {
			if _, ok := connections[connection.ID()]; ok {
				rooms = append(rooms, room)
			}
		}
	}
	return rooms
}

// Emit emits given connectionID, event & args to target clientID by connection
func (broadcast *broadcast) Emit(connectID string, event string, args ...interface{}) {
	broadcast.lock.RLock()
	defer broadcast.lock.RUnlock()

	if connection, ok := broadcast.connections[connectID]; ok {
		connection.Emit(event, args...)
	}
}
