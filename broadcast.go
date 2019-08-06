package socketio

import "sync"

// Broadcast is the adaptor to handle broadcasts & rooms for socket.io server API
type Broadcast interface {
	Join(room string, connection Conn)            // Join causes the connection to join a room
	Leave(room string, connection Conn)           // Leave causes the connection to leave a room
	LeaveAll(connection Conn)                     // LeaveAll causes given connection to leave all rooms
	Clear(room string)                            // Clear causes removal of all connections from the room
	Send(room, event string, args ...interface{}) // Send will send an event with args to the room
	SendAll(event string, args ...interface{})    // SendAll will send an event with args to all the rooms
	Len(room string) int                          // Len gives number of connections in the room
	Rooms(connection Conn) []string               // Gives list of all the rooms if no connection given, else list of all the rooms the connection joined
}

// broadcast gives Join, Leave & BroadcastTO server API support to socket.io along with room management
type broadcast struct {
	rooms map[string]map[string]Conn // map of rooms where each room contains a map of connection id to connections in that room
	lock  sync.RWMutex               // access lock for rooms
}

// NewBroadcast creates a new broadcast adapter
func NewBroadcast() Broadcast {
	return &broadcast{rooms: make(map[string]map[string]Conn)}
}

// Join joins the given connection to the broadcast room
func (broadcast *broadcast) Join(room string, connection Conn) {
	// get write lock
	broadcast.lock.Lock()
	defer broadcast.lock.Unlock()

	// check if room already has connection mappings, if not then create one
	if _, ok := broadcast.rooms[room]; !ok {
		broadcast.rooms[room] = make(map[string]Conn)
	}

	// add the connection to the rooms connection map
	broadcast.rooms[room][connection.ID()] = connection
}

// Leave leaves the given connection from given room (if exist)
func (broadcast *broadcast) Leave(room string, connection Conn) {
	// get write lock
	broadcast.lock.Lock()
	defer broadcast.lock.Unlock()

	// check if rooms connection
	if connections, ok := broadcast.rooms[room]; ok {
		// remove the connection from the room
		delete(connections, connection.ID())

		// check if no more connection is left to the room, then delete the room
		if len(connections) == 0 {
			delete(broadcast.rooms, room)
		}
	}
}

// LeaveAll leaves the given connection from all rooms
func (broadcast *broadcast) LeaveAll(connection Conn) {
	// get write lock
	broadcast.lock.Lock()
	defer broadcast.lock.Unlock()

	// iterate through each room
	for room, connections := range broadcast.rooms {
		// remove the connection from the rooms connections
		delete(connections, connection.ID())

		// check if no more connection is left to the room, then delete the room
		if len(connections) == 0 {
			delete(broadcast.rooms, room)
		}
	}
}

// Clear clears the room
func (broadcast *broadcast) Clear(room string) {
	// get write lock
	broadcast.lock.Lock()
	defer broadcast.lock.Unlock()

	// delete the room
	delete(broadcast.rooms, room)
}

// Send sends given event & args to all the connections in the specified room
func (broadcast *broadcast) Send(room, event string, args ...interface{}) {
	// get a read lock
	broadcast.lock.RLock()
	defer broadcast.lock.RUnlock()

	// iterate through each connection in the room
	for _, connection := range broadcast.rooms[room] {
		// emit the event to the connection
		connection.Emit(event, args...)
	}
}

// SendAll sends given event & args to all the connections to all the rooms
func (broadcast *broadcast) SendAll(event string, args ...interface{}) {
	// get a read lock
	broadcast.lock.RLock()
	defer broadcast.lock.RUnlock()

	// iterate through each room
	for _, connections := range broadcast.rooms {
		// iterate through each connection in the room
		for _, connection := range connections {
			// emit the event to the connection
			connection.Emit(event, args...)
		}
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

	if connection == nil {
		// create a new list of all the room names
		rooms := make([]string, 0)

		// iterate through the rooms map and add the room name to the above list
		for room := range broadcast.rooms {
			rooms = append(rooms, room)
		}

		return rooms
	} else {
		// create a new list of all the room names the connection is joined to
		rooms := make([]string, 0)

		// iterate through the rooms map and add the room name to the above list
		for room, connections := range broadcast.rooms {
			// check if the connection is joined to the room
			if _, ok := connections[connection.ID()]; ok {
				// add the room name to the list
				rooms = append(rooms, room)
			}
		}

		return rooms
	}
}
