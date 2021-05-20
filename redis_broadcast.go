package socketio

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/gomodule/redigo/redis"
)

// RedisAdapterOptions is configuration to create new adapter
type RedisAdapterOptions struct {
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

	requests map[string]interface{}

	rooms map[string]map[string]Conn

	lock sync.RWMutex
}

// request types
const (
	roomLenReqType   = "0"
	clearRoomReqType = "1"
	allRoomReqType   = "2"
)

// request structs
type roomLenRequest struct {
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

type allRoomRequest struct {
	RequestType string
	RequestID   string
	rooms       map[string]bool `json:"-"`
	numSub      int             `json:"-"`
	msgCount    int             `json:"-"`
	mutex       sync.Mutex      `json:"-"`
	done        chan bool       `json:"-"`
}

// response struct
type roomLenResponse struct {
	RequestType string
	RequestID   string
	Connections int
}

type allRoomResponse struct {
	RequestType string
	RequestID   string
	Rooms       []string
}

func newRedisBroadcast(nsp string, adapter *RedisAdapterOptions) (*redisBroadcast, error) {
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
		return nil, err
	}

	sub, err := redis.Dial("tcp", redisAddr)
	if err != nil {
		return nil, err
	}

	bc.pub = redis.PubSubConn{Conn: pub}
	bc.sub = redis.PubSubConn{Conn: sub}

	bc.nsp = nsp
	bc.uid = newV4UUID()
	bc.key = bc.prefix + "#" + bc.nsp + "#" + bc.uid
	bc.reqChannel = bc.prefix + "-request#" + bc.nsp
	bc.resChannel = bc.prefix + "-response#" + bc.nsp
	bc.requests = make(map[string]interface{})

	if err = bc.sub.PSubscribe(bc.prefix + "#" + bc.nsp + "#*"); err != nil {
		return nil, err
	}

	if err = bc.sub.Subscribe(bc.reqChannel, bc.resChannel); err != nil {
		return nil, err
	}

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

				err = bc.onMessage(m.Channel, m.Data)
				if err != nil {
					return
				}

			case redis.Subscription:
				if m.Count == 0 {
					return
				}

			case error:
				return
			}
		}
	}()

	return &bc, nil
}

// AllRooms gives list of all rooms available for redisBroadcast.
func (bc *redisBroadcast) AllRooms() []string {
	req := allRoomRequest{
		RequestType: allRoomReqType,
		RequestID:   newV4UUID(),
	}
	reqJSON, _ := json.Marshal(&req)

	req.rooms = make(map[string]bool)
	numSub, _ := bc.getNumSub(bc.reqChannel)
	req.numSub = numSub
	req.done = make(chan bool, 1)

	bc.requests[req.RequestID] = &req
	_, err := bc.pub.Conn.Do("PUBLISH", bc.reqChannel, reqJSON)
	if err != nil {
		return []string{} // if error occurred,return empty
	}

	<-req.done

	rooms := make([]string, 0, len(req.rooms))
	for room := range req.rooms {
		rooms = append(rooms, room)
	}

	delete(bc.requests, req.RequestID)
	return rooms
}

// Join joins the given connection to the redisBroadcast room.
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

// LeaveAll leaves the given connection from all rooms.
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

// Clear clears the room.
func (bc *redisBroadcast) Clear(room string) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	delete(bc.rooms, room)
	go bc.publishClear(room)
}

// Send sends given event & args to all the connections in the specified room.
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

// SendAll sends given event & args to all the connections to all the rooms.
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

// ForEach sends data returned by DataFunc, if room does not exits sends nothing.
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

// Len gives number of connections in the room.
func (bc *redisBroadcast) Len(room string) int {
	req := roomLenRequest{
		RequestType: roomLenReqType,
		RequestID:   newV4UUID(),
		Room:        room,
	}

	reqJSON, err := json.Marshal(&req)
	if err != nil {
		return -1
	}

	numSub, err := bc.getNumSub(bc.reqChannel)
	if err != nil {
		return -1
	}

	req.numSub = numSub

	req.done = make(chan bool, 1)

	bc.requests[req.RequestID] = &req
	_, err = bc.pub.Conn.Do("PUBLISH", bc.reqChannel, reqJSON)
	if err != nil {
		return -1
	}

	<-req.done

	delete(bc.requests, req.RequestID)
	return req.connections
}

// Rooms gives the list of all the rooms available for redisBroadcast in case of
// no connection is given, in case of a connection is given, it gives
// list of all the rooms the connection is joined to.
func (bc *redisBroadcast) Rooms(connection Conn) []string {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	if connection == nil {
		return bc.AllRooms()
	}

	return bc.getRoomsByConn(connection)
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
	if !ok {
		return errors.New("invalid event")
	}

	if room != "" {
		bc.send(room, event, args...)
	} else {
		bc.sendAll(event, args...)
	}

	return nil
}

// Get the number of subscribers of a channel.
func (bc *redisBroadcast) getNumSub(channel string) (int, error) {
	rs, err := bc.pub.Conn.Do("PUBSUB", "NUMSUB", channel)
	if err != nil {
		return 0, err
	}

	numSub64, ok := rs.([]interface{})[1].(int)
	if !ok {
		return 0, errors.New("redis reply cast to int error")
	}
	return numSub64, nil
}

// Handle request from redis channel.
func (bc *redisBroadcast) onRequest(msg []byte) {
	var req map[string]string

	if err := json.Unmarshal(msg, &req); err != nil {
		return
	}

	var res interface{}
	switch req["RequestType"] {
	case roomLenReqType:
		res = roomLenResponse{
			RequestType: req["RequestType"],
			RequestID:   req["RequestID"],
			Connections: len(bc.rooms[req["Room"]]),
		}
		bc.publish(bc.resChannel, &res)

	case allRoomReqType:
		res := allRoomResponse{
			RequestType: req["RequestType"],
			RequestID:   req["RequestID"],
			Rooms:       bc.allRooms(),
		}
		bc.publish(bc.resChannel, &res)

	case clearRoomReqType:
		if bc.uid == req["UUID"] {
			return
		}
		bc.clear(req["Room"])

	default:
	}
}

func (bc *redisBroadcast) publish(channel string, msg interface{}) {
	resJSON, err := json.Marshal(msg)
	if err != nil {
		return
	}

	_, err = bc.pub.Conn.Do("PUBLISH", channel, resJSON)
	if err != nil {
		return
	}
}

// Handle response from redis channel.
func (bc *redisBroadcast) onResponse(msg []byte) {
	var res map[string]interface{}

	err := json.Unmarshal(msg, &res)
	if err != nil {
		return
	}

	req, ok := bc.requests[res["RequestID"].(string)]
	if !ok {
		return
	}

	switch res["RequestType"] {
	case roomLenReqType:
		roomLenReq := req.(*roomLenRequest)

		roomLenReq.mutex.Lock()
		roomLenReq.msgCount++
		roomLenReq.connections += int(res["Connections"].(float64))
		roomLenReq.mutex.Unlock()

		if roomLenReq.numSub == roomLenReq.msgCount {
			roomLenReq.done <- true
		}

	case allRoomReqType:
		allRoomReq := req.(*allRoomRequest)
		rooms, ok := res["Rooms"].([]interface{})
		if !ok {
			allRoomReq.done <- true
			return
		}

		allRoomReq.mutex.Lock()
		allRoomReq.msgCount++
		for _, room := range rooms {
			allRoomReq.rooms[room.(string)] = true
		}
		allRoomReq.mutex.Unlock()

		if allRoomReq.numSub == allRoomReq.msgCount {
			allRoomReq.done <- true
		}

	default:
	}
}

func (bc *redisBroadcast) publishClear(room string) {
	req := clearRoomRequest{
		RequestType: clearRoomReqType,
		RequestID:   newV4UUID(),
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

func (bc *redisBroadcast) send(room string, event string, args ...interface{}) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	connections, ok := bc.rooms[room]
	if !ok {
		return
	}

	for _, connection := range connections {
		connection.Emit(event, args...)
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
	bcMessageJSON, err := json.Marshal(bcMessage)
	if err != nil {
		return
	}

	_, err = bc.pub.Conn.Do("PUBLISH", bc.key, bcMessageJSON)
	if err != nil {
		return
	}
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

func (bc *redisBroadcast) allRooms() []string {
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
