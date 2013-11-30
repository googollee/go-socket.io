socket.io library for Golang

forked from [http://code.google.com/p/go-socketio](http://code.google.com/p/go-socketio)

Added a socket.io client for quick use
Fixed the disconnect event
Added persistent sessionIds
Added session values


Demo
##

*server:*

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
    fmt.Printf("in %s, title: %s, body: %s, article number: %i", ns.Endpoint(), title, body, article_num)
}

func onConnect(ns *socketio.NameSpace) {
    fmt.Println("connected:", ns.Id(), " in channel ", ns.Endpoint())
    ns.Call("news", "abc", 3)
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

    //in channel abc
    sio.Of("/abc").On("connect", onConnect)
    sio.Of("/abc").On("disconnect", onDisconnect)
    sio.Of("/abc").On("news", news)

    //this will serve a http static file server
    sio.Handle("/", http.FileServer(http.Dir("./public/")))
    //startup the server
    log.Fatal(http.ListenAndServe(":3000", sio))

    fmt.Println("end")
}
```

*go client:*

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
      ns.Call("news", "this is title", "this is body", 1)
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

*javascript client*

 *NOTE:* There is a provided socket.io.js file in the lib folder for including in your project

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

