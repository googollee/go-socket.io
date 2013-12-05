package main

import (
	"github.com/googollee/go-socket.io"
	"log"
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
