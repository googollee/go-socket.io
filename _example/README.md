## Simple example

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/googollee/go-socket.io"
)

func main() {
	server, serveError  := socketio.NewServer(nil)
	if serveError != nil {
		log.Fatalln(serveError)
	}
	
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected:", s.ID())
		return nil
	})

	server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		fmt.Println("notice:", msg)
		s.Emit("reply", "have "+msg)
	})

	server.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
		s.SetContext(msg)
		return "recv " + msg
	})

	server.OnEvent("/", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		s.Emit("bye", last)
		s.Close()
		return last
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		fmt.Println("meet error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Println("closed", reason)
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving at localhost:8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
```

## Acknowledgements in go-socket.io 1.X.X

[See documentation about acknowledgements](http://socket.io/docs/#sending-and-getting-data-(acknowledgements)

##### Sending ACK with data from SERVER to CLIENT

* Client-side

```javascript
 //using client-side socket.io-1.X.X.js
 socket.emit('some:event', JSON.stringify(someData), function(data){
       console.log('ACK from server wtih data: ', data));
 });
```

* Server-side

```go
// The return type may vary depending on whether you will return
// In golang implementation of Socket.IO don't used callbacks for acknowledgement,
// but used return value, which wrapped into ack package and returned to the client's callback in JavaScript
so.On("some:event", func(msg string) string {
	return msg //Sending ack with data in msg back to client, using "return statement"
})
```

##### Sending ACK with data from CLIENT to SERVER

* Client-side

```javascript
//using client-side socket.io-1.X.X.js
//last parameter of "on" handler is callback for sending ack to server with data or without data
socket.on('some:event', function (msg, sendAckCb) {
    //Sending ACK with data to server after receiving some:event from server
    sendAckCb(JSON.stringify(data)); // for example used serializing to JSON
}
```

* Server-side

```go
//You can use Emit or BroadcastTo with last parameter as callback for handling ack from client
//Sending packet to room "room_name" and event "some:event"
so.BroadcastTo("room_name", "some:event", dataForClient, func (so socketio.Socket, data string) {
	log.Println("Client ACK with data: ", data)
})

// Or

so.Emit("some:event", dataForClient, func (so socketio.Socket, data string) {
	log.Println("Client ACK with data: ", data)
})
```

##### Broadcast to All connected Client
* Server-side

```go
//Add all connected user to a room, in example? "bcast"
server.OnConnect("/", func(s socketio.Conn) error {
	s.SetContext("")
	fmt.Println("connected:", s.ID())
	s.Join("bcast")
	return nil
})

//Broadcast message to all connected user
server.BroadcastToRoom("", "bcast", "event:name", msg)
```
* Client-side
```
socket.on('some:event', function (msg) {
	console.log(msg);
});
```


##### Catch Disconnected reason

* Server-side

```go

so.OnDisconnect("/", func(so socketio.Conn, reason string) {
  	log.Println("closed", reason)
})
```

Possible reasons:


| Reason | Side | Description |
|------------|-------------|------------|
| client namespace disconnect | Client Side | Got disconnect packet from client |
