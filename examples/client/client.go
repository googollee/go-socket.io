package main

import (
  "log"
  "github.com/tanema/go-socket.io"
)

func client() {
  client, err := socketio.Dial("http://127.0.0.1:3000", "http://127.0.0.1:3000")
  if err != nil {
    panic(err)
  }
  client.On("connect", func(ns *socketio.NameSpace) {
    log.Println("connected")
    ns.Emit("ping", "I pinged")
  })
  client.On("news", func(ns *socketio.NameSpace, message string) {
    log.Println(message)
  })
  client.On("pong", func(ns *socketio.NameSpace, message string) {
    log.Println("got pong: ", message)
  })
  client.Run()
}

func pol() {
  client, err := socketio.Dial("http://127.0.0.1:3000/pol", "http://127.0.0.1:3000")
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
  go client()
  pol()
}
