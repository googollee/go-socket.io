# socket.io

[![GoDoc](http://godoc.org/github.com/googollee/go-socket.io?status.svg)](http://godoc.org/github.com/googollee/go-socket.io) [![Build Status](https://travis-ci.org/googollee/go-socket.io.svg)](https://travis-ci.org/googollee/go-socket.io)

go-socket.io is an implementation of [socket.io](http://socket.io) in golang, which is a realtime application framework.

It is compatible with latest implementation of socket.io in node.js, and supports room and namespace.

* for compatability with socket.io 0.9.x, please use branch 0.9.x *

## Install

Install the package with:

```bash
go get github.com/googollee/go-socket.io
```

Import it with:

```go
import "github.com/googollee/go-socket.io"
```

and use `socketio` as the package name inside the code.

## Example

Please check the example folder for details.

```go
package main

import (
	"log"
	"net/http"

	"github.com/googollee/go-socket.io"
)

func main() {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.On("connection", func(so socketio.Socket) {
		log.Println("on connection")
		so.Join("chat")
		so.On("chat message", func(msg string) {
			log.Println("emit:", so.Emit("chat message", msg))
			so.BroadcastTo("chat", "chat message", msg)
		})
		so.On("disconnection", func() {
			log.Println("on disconnect")
		})
	})
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving at localhost:5000...")
	log.Fatal(http.ListenAndServe(":5000", nil))
}
```

## Example of event handler with acknowledgement response

[See documentation about acknowledgements](http://socket.io/docs/#sending-and-getting-data-(acknowledgements))

```go
// The return type may vary depending on whether you will return
// In golang implementation of socket.io don't used callbacks for acknowledgement,
// but used return value, which wrapped into ack package and returned to the client's callback in JavaScript
so.On("chat message withack", func(msg string) string {
	return msg
})
```

## License

The 3-clause BSD License  - see LICENSE for more details
