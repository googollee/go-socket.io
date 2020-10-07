package socketio

import "sync"

// EachFunc typed for each callback function
type EachFunc func(Conn)

// Broadcast is the adaptor to handle broadcasts & rooms for socket.io server API
type Broadcast interface {
	Join(room string, connection Conn)            // Join causes the connection to join a room
	Leave(room string, connection Conn)           // Leave causes the connection to leave a room
	LeaveAll(connection Conn)                     // LeaveAll causes given connection to leave all rooms
	Clear(room string)                            // Clear causes removal of all connections from the room
	Send(room, event string, args ...interface{}) // Send will send an event with args to the room
	SendAll(event string, args ...interface{})    // SendAll will send an event with args to all the rooms
	ForEach(room string, f EachFunc)              // ForEach sends data by DataFunc, if room does not exits sends nothing
	Len(room string) int                          // Len gives number of connections in the room
	Rooms(connection Conn) []string               // Gives list of all the rooms if no connection given, else list of all the rooms the connection joined
	AllRooms() []string                           // Gives list of all the rooms the connection joined
}

// broadcast gives Join, Leave & BroadcastTO server API support to socket.io along with room management
// map of rooms where each room contains a map of connection id to connections in that room
type broadcast struct {
	rooms map[string]map[string]Conn

	lock sync.RWMutex
}

// newBroadcast creates a new broadcast adapter
func newBroadcast() *broadcast {
	return &broadcast{
		rooms: make(map[string]map[string]Conn),
	}
}

// Join joins the given connection to the broadcast room
func (bc *broadcast) Join(room string, connection Conn) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	if _, ok := bc.rooms[room]; !ok {
		bc.rooms[room] = make(map[string]Conn)
	}

	bc.rooms[room][connection.ID()] = connection
}

// Leave leaves the given connection from given room (if exist)
func (bc *broadcast) Leave(room string, connection Conn) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	if connections, ok := bc.rooms[room]; ok {
		delete(connections, connection.ID())

		if len(connections) == 0 {
			delete(bc.rooms, room)
		}
	}
}

// LeaveAll leaves the given connection from all rooms
func (bc *broadcast) LeaveAll(connection Conn) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	for room, connections := range bc.rooms {
		delete(connections, connection.ID())

		if len(connections) == 0 {
			delete(bc.rooms, room)
		}
	}
}

// Clear clears the room
func (bc *broadcast) Clear(room string) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	delete(bc.rooms, room)
}

// Send sends given event & args to all the connections in the specified room
func (bc *broadcast) Send(room, event string, args ...interface{}) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	for _, connection := range bc.rooms[room] {
		connection.Emit(event, args...)
	}
}

// SendAll sends given event & args to all the connections to all the rooms
func (bc *broadcast) SendAll(event string, args ...interface{}) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	for _, connections := range bc.rooms {
		for _, connection := range connections {
			connection.Emit(event, args...)
		}
	}
}

// ForEach sends data returned by DataFunc, if room does not exits sends nothing
func (bc *broadcast) ForEach(room string, f EachFunc) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	occupants, ok := bc.rooms[room]
	if !ok {
		return
	}

	for _, connection := range occupants {
		f(connection)
	}
}

// Len gives number of connections in the room
func (bc *broadcast) Len(room string) int {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	return len(bc.rooms[room])
}

// Rooms gives the list of all the rooms available for broadcast in case of
// no connection is given, in case of a connection is given, it gives
// list of all the rooms the connection is joined to
func (bc *broadcast) Rooms(connection Conn) []string {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	if connection == nil {
		return bc.AllRooms()
	}

	return bc.getRoomsByConn(connection)
}

// AllRooms gives list of all rooms available for broadcast
func (bc *broadcast) AllRooms() []string {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	rooms := make([]string, 0, len(bc.rooms))
	for room := range bc.rooms {
		rooms = append(rooms, room)
	}

	return rooms
}

func (bc *broadcast) getRoomsByConn(connection Conn) []string {
	var rooms []string

	for room, connections := range bc.rooms {
		if _, ok := connections[connection.ID()]; ok {
			rooms = append(rooms, room)
		}
	}

	return rooms
}
