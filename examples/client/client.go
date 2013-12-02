package main

import (
  "log"
  "github.com/tanema/go-socket.io"
)

func client() {
  client, err := socketio.Dial("http://127.0.0.1:3000")
  if err != nil {
    panic(err)
  }
  client.On("connect", func(ns *socketio.NameSpace) {
    log.Println("connected")
    ns.Emit("ping", nil)
  })
  client.On("news", func(ns *socketio.NameSpace, message string) {
    log.Println(message)
  })
  client.On("pong", func(ns *socketio.NameSpace) {
    log.Println("got pong")
  })
  client.Run()
}

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
  go client()
  pol()
}
