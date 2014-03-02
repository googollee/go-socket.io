##socket.io library for Golang

forked from [http://code.google.com/p/go-socketio](http://code.google.com/p/go-socketio)

##Demo

**server:**

```go
package main

import (
  "fmt"
  "github.com/googollee/go-socket.io"
  "log"
  "net/http"
)

func news(ns *socketio.NameSpace, title, body string, article_num int) {
  var name string
  name = ns.Session.Values["name"].(string)
  fmt.Printf("%s said in %s, title: %s, body: %s, article number: %i", name, ns.Endpoint(), title, body, article_num)
}

func onConnect(ns *socketio.NameSpace) {
  fmt.Println("connected:", ns.Id(), " in channel ", ns.Endpoint())
  ns.Session.Values["name"] = "this guy"
  ns.Emit("news", "this is totally news", 3)
}

func onDisconnect(ns *socketio.NameSpace) {
  fmt.Println("disconnected:", ns.Id(), " in channel ", ns.Endpoint())
}

func main() {
  sock_config := &socketio.Config{}
  sock_config.HeartbeatTimeout = 2
  sock_config.ClosingTimeout = 4

  sio := socketio.NewSocketIOServer(sock_config)

  // Handler for new connections, also adds socket.io event handlers
  sio.On("connect", onConnect)
  sio.On("disconnect", onDisconnect)
  sio.On("news", news)
  sio.On("ping", func(ns *socketio.NameSpace){
    sio.Broadcast("pong", nil)
  })

  //in politics channel
  sio.Of("/pol").On("connect", onConnect)
  sio.Of("/pol").On("disconnect", onDisconnect)
  sio.Of("/pol").On("news", news)
  sio.Of("/pol").On("ping", func(ns *socketio.NameSpace){
    sio.In("/pol").Broadcast("pong", nil)
  })

  //this will serve a http static file server
  sio.Handle("/", http.FileServer(http.Dir("./public/")))
  //startup the server
  log.Fatal(http.ListenAndServe(":3000", sio))
}
```

**go client:**

```go
package main

import (
  "log"
  "github.com/googollee/go-socket.io"
)

func pol() {
  client, err := socketio.Dial("http://127.0.0.1:3000/pol")
  if err != nil {
    panic(err)
  }
  client.On("connect", func(ns *socketio.NameSpace) {
    log.Println("pol connected")
  })
  client.On("news", func(ns *socketio.NameSpace, message string) {
    log.Println(message, " in Pol")
  })
  client.Run()
}

func main() {
  client, err := socketio.Dial("http://127.0.0.1:3000/")
  if err != nil {
    panic(err)
  }
  client.On("connect", func(ns *socketio.NameSpace) {
    log.Println("connected")
    ns.Emit("ping", nil)
  })
  client.Of("/pol").On("news", func(ns *socketio.NameSpace, message string) {
    log.Println(message, " in Pol 2")
  })
  client.On("news", func(ns *socketio.NameSpace, message string) {
    log.Println(message)
  })
  client.On("pong", func(ns *socketio.NameSpace) {
    log.Println("got pong")
  })

  go pol()

  client.Run()
}
``` 

**javascript client**

 **NOTE:** There is a provided socket.io.js file in the lib folder for including in your project

```javascript
  var socket = io.connect();
  socket.on("connect", function(){
    socket.emit("news", "this is title", "this is body", 1)
  })
  socket.on("news", function(message, urgency){
    console.log(message + urgency);
    socket.emit("ping")
  })
  socket.on("pong", function() {
    console.log("got pong")
  })
  socket.of("/pol").on("news", function(message, urgency){
    console.log(message + urgency);
    socket.emit("ping")
  })
  socket.of("/pol").on("pong", function() {
    console.log("got pong")
  })
  socket.on("disconnect", function() {
    alert("You have disconnected from the server")
  })
  var pol = io.connect("http://localhost/pol");
  pol.on("pong", function() {
    console.log("got pong from pol")
  })
  pol.on("news", function(message, urgency){
    console.log(message + urgency);
    socket.emit("ping")
  })
```

##Changelog
- Added a socket.io client for quick use
- Fixed the disconnect event
- Added persistent sessionIds
- Added session values
- Added broadcast
- Added a simpler Emit function to namespaces
- Fixed connected event on endpoints
- Added events without arguments
- Fixed go client endpoints
