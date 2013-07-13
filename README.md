socket.io library for Golang

forked from [http://code.google.com/p/go-socketio](http://code.google.com/p/go-socketio)

simple demo:

```go
package main

import (
    "fmt"
    "github.com/googollee/go-socketio"
    "log"
    "net/http"
)

func ferret(ns *socketio.NameSpace, a string, i int) string {
    fmt.Println(a, i)
    return "woot"
}

func event(ns *socketio.NameSpace, data struct{ My string }) {
    fmt.Println("event:", data.My)
}

func news(ns *socketio.NameSpace, arg map[string]string) (int, string) {
    fmt.Printf("in news, name: %s, args: %#v\n", ns.Name, arg)
    return 1, "str"
}

func onConnect(ns *socketio.NameSpace) string {
    fmt.Println("connected:", ns.Name)
    ns.Call("news", nil, "abc")
    return "news"
}

func main() {
    sock_config := &socketio.Config{}
    sock_config.HeartbeatTimeout = 2
    sock_config.ClosingTimeout = 4

    sio := socketio.NewSocketIOServer(sock_config)

    // Handler for new connections, also adds socket.io event handlers
    err := sio.On("connect", onConnect)
    fmt.Println(err)
    err = sio.On("disconnect", func(ns *socketio.NameSpace) { fmt.Println("Disconnect!", ns.Name) })
    fmt.Println(err)
    err = sio.On("news", news)
    fmt.Println(err)
    err = sio.On("my other event", event)
    fmt.Println(err)
    err = sio.On("ferret", ferret)
    fmt.Println(err)

    sio.Of("/abc").On("connect", onConnect)
    sio.Of("/abc").On("news", news)
    sio.Of("/abc").On("disconnect", func(ns *socketio.NameSpace) { fmt.Println("Disconnect!", ns.Name) })
    sio.Of("/abc").On("my other event", event)
    sio.Of("/abc").On("ferret", ferret)

    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        sio.ServeHTTP(w, r)
    })
    log.Fatal(http.ListenAndServe(":8080", mux))

    fmt.Println("end")
}
```

client:

```html
<script src="/socket.io/socket.io.js"></script>
<script>
  var socket = io.connect('http://localhost:8080');
  socket.emit('news', { my: 'data' }, function (i, s) {
    console.log(i, s);
  });
  socket.on('news', function (data) {
    console.log(data);
    socket.emit('my other event', { my: 'data' });
    socket.emit('ferret', 'tobi', 123, function (data) {
      console.log(data); // data will be 'woot'
    });
    socket.emit('ferret', 'tobi', 123, function (data) {
      console.log(data); // data will be 'woot'
    });
  });
</script>
``` 