package socketio

import (
	"errors"
	"log"
	"os"
	"strings"
	"sync"

	"encoding/json"

	"github.com/gomodule/redigo/redis"
	uuid "github.com/satori/go.uuid"
)

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
	host   string
	port   string
	prefix string

	pub redis.PubSubConn
	sub redis.PubSubConn

	nsp        string
	uid        string
	key        string
	reqChannel string
	resChannel string

	requets map[string]interface{}

	rooms map[string]map[string]Conn

	lock sync.RWMutex
}

// request types
const (
	clientsReqType   = "0"
	clearRoomReqType = "1"
)

// request structs
type clientsRequest struct {
	RequestType string
	RequestID   string
	Room        string
	numSub      int        `json:"-"`
	msgCount    int        `json:"-"`
	connections int        `json:"-"`
	mutex       sync.Mutex `json:"-"`
	done        chan bool  `json:"-"`
}

type clearRoomRequest struct {
	RequestType string
	RequestID   string
	Room        string
	UUID        string
}

// response struct
type clientsResponse struct {
	RequestType string
	RequestID   string
	Connections int
}

// newBroadcast creates a new broadcast adapter
func newBroadcast(nsp string) *broadcast {
	bc := broadcast{
		rooms: make(map[string]map[string]Conn),
	}

	bc.host = os.Getenv("REDIS_HOST")
	if bc.host == "" {
		bc.host = "127.0.0.1"
	}

	bc.port = os.Getenv("REDIS_PORT")
	if bc.port == "" {
		bc.port = "6379"
	}

	bc.prefix = os.Getenv("SOCKET_PREFIX")
	if bc.prefix == "" {
		bc.prefix = "socket.io"
	}

	redisAddr := bc.host + ":" + bc.port
	// log.Println("redis address:", redisAddr)
	pub, err := redis.Dial("tcp", redisAddr)
	if err != nil {
		panic(err)
	}
	sub, err := redis.Dial("tcp", redisAddr)
	if err != nil {
		panic(err)
	}

	bc.pub = redis.PubSubConn{Conn: pub}
	bc.sub = redis.PubSubConn{Conn: sub}

	bc.nsp = nsp
	bc.uid = uuid.NewV4().String()
	bc.key = bc.prefix + "#" + bc.nsp + "#" + bc.uid
	bc.reqChannel = bc.prefix + "-request#" + bc.nsp
	bc.resChannel = bc.prefix + "-response#" + bc.nsp
	bc.requets = make(map[string]interface{})

	log.Println("bc key:", bc.key)

	bc.sub.PSubscribe(bc.prefix + "#" + bc.nsp + "#*")
	bc.sub.Subscribe(bc.reqChannel, bc.resChannel)

	go func() {
		for {
			switch m := bc.sub.Receive().(type) {
			case redis.Message:
				if m.Channel == bc.reqChannel {
					bc.onRequest(m.Data)
					break
				} else if m.Channel == bc.resChannel {
					bc.onResponse(m.Data)
					break
				}

				bc.onMessage(m.Channel, m.Data)
			case redis.Subscription:
				log.Printf("Subscription: %s %s %d\n", m.Kind, m.Channel, m.Count)
				if m.Count == 0 {
					return
				}
			case error:
				log.Printf("error: %v\n", m)
				return
			}
		}
	}()

	return &bc
}

func (bc *broadcast) onMessage(channel string, msg []byte) error {
	channelParts := strings.Split(channel, "#")
	nsp := channelParts[len(channelParts)-2]
	if bc.nsp != nsp {
		log.Println("different nsp")
		return nil
	}
	uid := channelParts[len(channelParts)-1]
	// log.Println("bc id:", bc.uid)
	// log.Println("uid:", uid)
	if bc.uid == uid {
		// log.Println("same uid")
		return nil
	}

	var bcMessage map[string][]interface{}
	err := json.Unmarshal(msg, &bcMessage)
	if err != nil {
		return errors.New("invalid broadcast message")
	}

	args := bcMessage["args"]
	opts := bcMessage["opts"]

	room, ok := opts[0].(string)
	// log.Println("room:", room)
	if !ok {
		log.Println("room is not a string")
	}

	event, ok := opts[1].(string)

	// log.Printf("Message: %s %s\n", channel, msg)
	bc.SendOnSubcribe(room, event, args...)
	return nil
}

// Get the number of subcribers of a channel
func (bc *broadcast) getNumSub(channel string) (int, error) {
	rs, err := bc.pub.Conn.Do("PUBSUB", "NUMSUB", channel)
	if err != nil {
		return 0, err
	}

	var numSub64 int64
	numSub64 = rs.([]interface{})[1].(int64)
	return int(numSub64), nil
}

// Handle request from redis channel
func (bc *broadcast) onRequest(msg []byte) {
	var req map[string]string
	err := json.Unmarshal(msg, &req)
	if err != nil {
		log.Println("err on request:", err)
		return
	}
	log.Printf("on req: %s\n", req)

	var res interface{}
	switch req["RequestType"] {
	case clientsReqType:
		res = clientsResponse{
			RequestType: req["RequestType"],
			RequestID:   req["RequestID"],
			Connections: len(bc.rooms[req["Room"]]),
		}
		bc.publish(bc.resChannel, &res)

	case clearRoomReqType:
		if bc.uid == req["UUID"] {
			return
		}
		bc.clear(req["Room"])

	default:
		log.Println("unknown reuqest")
		return
	}

}

func (bc *broadcast) publish(channel string, msg interface{}) {
	resJSON, _ := json.Marshal(msg)
	log.Printf("publish msg: %s\n", resJSON)
	bc.pub.Conn.Do("PUBLISH", channel, resJSON)
}

// Handle response from redis channel
func (bc *broadcast) onResponse(msg []byte) {
	var res map[string]interface{}
	err := json.Unmarshal(msg, &res)
	if err != nil {
		log.Println("on response:", err)
		return
	}

	req, ok := bc.requets[res["RequestID"].(string)]
	if !ok {
		return
	}

	log.Printf("on resp: %s\n", res)
	// log.Println("saved req:", req)
	switch res["RequestType"] {
	case clientsReqType:
		cReq := req.(*clientsRequest)

		cReq.mutex.Lock()
		// bc.lock.Lock()
		cReq.msgCount++
		cReq.connections += int(res["Connections"].(float64))
		// bc.lock.Unlock()
		cReq.mutex.Unlock()
		if cReq.numSub == cReq.msgCount {
			cReq.done <- true
		}

	// case clieclientRoomsReqType:

	default:
		log.Println("unknown response")
		return
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
	go bc.publishClear(room)
}

func (bc *broadcast) publishClear(room string) {
	req := clearRoomRequest{
		RequestType: clearRoomReqType,
		RequestID:   uuid.NewV4().String(),
		Room:        room,
		UUID:        bc.uid,
	}

	bc.publish(bc.reqChannel, &req)
}

func (bc *broadcast) clear(room string) {
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

	// bc.lock.RUnlock()

	opts := make([]interface{}, 2)
	opts[0] = room
	opts[1] = event

	bcMessage := map[string][]interface{}{
		"opts": opts,
		"args": args,
	}

	bcMessageJSON, _ := json.Marshal(bcMessage)

	bc.pub.Conn.Do("PUBLISH", bc.key, bcMessageJSON)
}

func (bc *broadcast) SendOnSubcribe(room string, event string, args ...interface{}) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	connections, ok := bc.rooms[room]
	if ok {
		for _, connection := range connections {
			connection.Emit(event, args...)
		}
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
	// bc.lock.RLock()
	// defer bc.lock.RUnlock()

	req := clientsRequest{
		RequestType: clientsReqType,
		RequestID:   uuid.NewV4().String(),
		Room:        room,
	}

	reqJSON, _ := json.Marshal(&req)
	log.Printf("req json: %s\n", reqJSON)

	numSub, _ := bc.getNumSub(bc.reqChannel)
	req.numSub = numSub
	req.done = make(chan bool)

	bc.requets[req.RequestID] = &req
	bc.pub.Conn.Do("PUBLISH", bc.reqChannel, reqJSON)
	<-req.done

	return req.connections
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
