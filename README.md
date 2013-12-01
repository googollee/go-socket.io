##socket.io library for Golang

forked from [http://code.google.com/p/go-socketio](http://code.google.com/p/go-socketio)

**this forks improvments**
- Added a socket.io client for quick use
- Fixed the disconnect event
- Added persistent sessionIds
- Added session values
- Added broadcast
- Added a simpler Emit function to namespaces
- Fixed connected event on endpoints

**TODO**
- double events on go client
- add fully functional examples
- events without arguments still work


##Demo

**server:**

```go
package main

import (
  "fmt"
  "github.com/tanema/go-socket.io"
  "log"
  "net/http"
  "time"
)

func news(ns *socketio.NameSpace, title, body string, article_num int) {
  var name string
  name = ns.Session.Values["name"].(string)
  fmt.Printf("%s said in %s, title: %s, body: %s, article number: %i", name, ns.Endpoint(), title, body, article_num)
}

func onConnect(ns *socketio.NameSpace) {
  fmt.Println("connected:", ns.Id(), " in channel ", ns.Endpoint())
  ns.Session.Values["name"] = "this guy"
  ns.Emit("news", "abc", 3)
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
  sio.On("ping", func(ns *socketio.NameSpace, message string){
    sio.Broadcast("pong", message)
  })

  //in channel abc
  sio.Of("/abc").On("connect", onConnect)
  sio.Of("/abc").On("disconnect", onDisconnect)
  sio.Of("/abc").On("news", news)
  sio.Of("/abc").On("ping", func(ns *socketio.NameSpace, message string){
    sio.In("/abc").Broadcast("pong", message)
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
  "os"
  "fmt"
  "github.com/tanema/go-socket.io"
  "time"
)

func main() {
  client, err := socketio.Dial("http://127.0.0.1:8080/", "http://127.0.0.1:8080")
  if err != nil {
    panic(err)
  }
  client.On("connect", func(ns *socketio.NameSpace) {
    ns.Emit("news", "this is title", "this is body", 1)
  })
  client.On("news", func(ns *socketio.NameSpace, message string, urgency int) { 
    fmt.Println("news", message, urgency) 
  })
  client.On("disconnect", func(ns *socketio.NameSpace) {
    os.Exit(1)
  })
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
  })
  socket.on("disconnect", function() {
    alert("You have disconnected from the server")
  })
```
