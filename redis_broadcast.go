package socketio

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"

	"github.com/gomodule/redigo/redis"
	uuid "github.com/satori/go.uuid"
)

// RedisAdapter is configuration to create new adapter
type RedisAdapter struct {
	Host   string
	Port   string
	Prefix string
}

// redisBroadcast gives Join, Leave & BroadcastTO server API support to socket.io along with room management
// map of rooms where each room contains a map of connection id to connections in that room
type redisBroadcast struct {
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

func newRedisBroadcast(nsp string, adapter *RedisAdapter) *redisBroadcast {
	bc := redisBroadcast{
		rooms: make(map[string]map[string]Conn),
	}

	bc.host = adapter.Host
	if bc.host == "" {
		bc.host = "127.0.0.1"
	}

	bc.port = adapter.Port
	if bc.port == "" {
		bc.port = "6379"
	}

	bc.prefix = adapter.Prefix
	if bc.prefix == "" {
		bc.prefix = "socket.io"
	}

	redisAddr := bc.host + ":" + bc.port
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
				if m.Count == 0 {
					return
				}
			case error:
				return
			}
		}
	}()

	return &bc
}

func (bc *redisBroadcast) onMessage(channel string, msg []byte) error {
	channelParts := strings.Split(channel, "#")
	nsp := channelParts[len(channelParts)-2]
	if bc.nsp != nsp {
		return nil
	}

	uid := channelParts[len(channelParts)-1]
	if bc.uid == uid {
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
	if !ok {
		return errors.New("invalid room")
	}

	event, ok := opts[1].(string)

	if room != "" {
		log.Println("receving msg:", args)
		bc.send(room, event, args...)
	} else {
		bc.sendAll(event, args...)
	}

	return nil
}

// Get the number of subcribers of a channel
func (bc *redisBroadcast) getNumSub(channel string) (int, error) {
	rs, err := bc.pub.Conn.Do("PUBSUB", "NUMSUB", channel)
	if err != nil {
		return 0, err
	}

	var numSub64 int64
	numSub64 = rs.([]interface{})[1].(int64)
	return int(numSub64), nil
}

// Handle request from redis channel
func (bc *redisBroadcast) onRequest(msg []byte) {
	var req map[string]string
	err := json.Unmarshal(msg, &req)
	if err != nil {
		return
	}

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
		return
	}
}

func (bc *redisBroadcast) publish(channel string, msg interface{}) {
	resJSON, _ := json.Marshal(msg)
	bc.pub.Conn.Do("PUBLISH", channel, resJSON)
}

// Handle response from redis channel
func (bc *redisBroadcast) onResponse(msg []byte) {
	var res map[string]interface{}
	err := json.Unmarshal(msg, &res)
	if err != nil {
		return
	}

	req, ok := bc.requets[res["RequestID"].(string)]
	if !ok {
		return
	}

	switch res["RequestType"] {
	case clientsReqType:
		cReq := req.(*clientsRequest)

		cReq.mutex.Lock()
		cReq.msgCount++
		cReq.connections += int(res["Connections"].(float64))
		cReq.mutex.Unlock()

		if cReq.numSub == cReq.msgCount {
			cReq.done <- true
		}

	default:
		return
	}
}

// Join joins the given connection to the redisBroadcast room
func (bc *redisBroadcast) Join(room string, connection Conn) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	if _, ok := bc.rooms[room]; !ok {
		bc.rooms[room] = make(map[string]Conn)
	}

	bc.rooms[room][connection.ID()] = connection
}

// Leave leaves the given connection from given room (if exist)
func (bc *redisBroadcast) Leave(room string, connection Conn) {
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
func (bc *redisBroadcast) LeaveAll(connection Conn) {
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
func (bc *redisBroadcast) Clear(room string) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	delete(bc.rooms, room)
	go bc.publishClear(room)
}

func (bc *redisBroadcast) publishClear(room string) {
	req := clearRoomRequest{
		RequestType: clearRoomReqType,
		RequestID:   uuid.NewV4().String(),
		Room:        room,
		UUID:        bc.uid,
	}

	bc.publish(bc.reqChannel, &req)
}

func (bc *redisBroadcast) clear(room string) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	delete(bc.rooms, room)
}

// Send sends given event & args to all the connections in the specified room
func (bc *redisBroadcast) Send(room, event string, args ...interface{}) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	connections, ok := bc.rooms[room]
	if ok {
		for _, connection := range connections {
			connection.Emit(event, args...)
		}
	}
	bc.publishMessage(room, event, args...)
}

func (bc *redisBroadcast) send(room string, event string, args ...interface{}) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	connections, ok := bc.rooms[room]
	if ok {
		for _, connection := range connections {
			connection.Emit(event, args...)
		}
	}
}

func (bc *redisBroadcast) publishMessage(room string, event string, args ...interface{}) {
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

// SendAll sends given event & args to all the connections to all the rooms
func (bc *redisBroadcast) SendAll(event string, args ...interface{}) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	for _, connections := range bc.rooms {
		for _, connection := range connections {
			connection.Emit(event, args...)
		}
	}
	bc.publishMessage("", event, args...)
}

func (bc *redisBroadcast) sendAll(event string, args ...interface{}) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	for _, connections := range bc.rooms {
		for _, connection := range connections {
			connection.Emit(event, args...)
		}
	}
}

// ForEach sends data returned by DataFunc, if room does not exits sends nothing
func (bc *redisBroadcast) ForEach(room string, f EachFunc) {
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
func (bc *redisBroadcast) Len(room string) int {
	// bc.lock.RLock()
	// defer bc.lock.RUnlock()

	req := clientsRequest{
		RequestType: clientsReqType,
		RequestID:   uuid.NewV4().String(),
		Room:        room,
	}

	reqJSON, _ := json.Marshal(&req)
	numSub, _ := bc.getNumSub(bc.reqChannel)
	req.numSub = numSub
	req.done = make(chan bool)

	bc.requets[req.RequestID] = &req
	bc.pub.Conn.Do("PUBLISH", bc.reqChannel, reqJSON)
	<-req.done

	return req.connections
}

// Rooms gives the list of all the rooms available for redisBroadcast in case of
// no connection is given, in case of a connection is given, it gives
// list of all the rooms the connection is joined to
func (bc *redisBroadcast) Rooms(connection Conn) []string {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	if connection == nil {
		return bc.AllRooms()
	}

	return bc.getRoomsByConn(connection)
}

// AllRooms gives list of all rooms available for redisBroadcast
func (bc *redisBroadcast) AllRooms() []string {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	rooms := make([]string, 0, len(bc.rooms))
	for room := range bc.rooms {
		rooms = append(rooms, room)
	}

	return rooms
}

func (bc *redisBroadcast) getRoomsByConn(connection Conn) []string {
	var rooms []string

	for room, connections := range bc.rooms {
		if _, ok := connections[connection.ID()]; ok {
			rooms = append(rooms, room)
		}
	}

	return rooms
}
