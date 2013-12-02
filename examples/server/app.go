package main

import (
	"net/http"
  "log"
  "github.com/tanema/go-socket.io"
)

func main() {
  sio := socketio.NewSocketIOServer(&socketio.Config{})
  sio.On("connect", func(ns *socketio.NameSpace){
    log.Println("Connected: ", ns.Id())
    sio.Broadcast("connected", ns.Id())
  })
  sio.On("disconnect", func(ns *socketio.NameSpace){
    log.Println("Disconnected: ", ns.Id())
    sio.Broadcast("disconnected", ns.Id())
  })
  sio.On("news", func(ns *socketio.NameSpace, message string){
    sio.Broadcast("news", message)
  })
  sio.On("ping", func(ns *socketio.NameSpace){
    ns.Emit("pong", nil)
  })
  sio.Of("/pol").On("connect", func(ns *socketio.NameSpace){
    log.Println("Pol Connected: ", ns.Id())
  })
  sio.Of("/pol").On("news", func(ns *socketio.NameSpace, message string){
    sio.In("/pol").Broadcast("news", message)
  })
  sio.Of("/pol").On("disconnect", func(ns *socketio.NameSpace){
    log.Println("Pol Disconnected: ", ns.Id())
  })
  sio.Handle("/", http.FileServer(http.Dir("./www/")))
	println("listening on port 3000")
  log.Fatal(http.ListenAndServe(":3000", sio))
}

