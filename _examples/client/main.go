package main

import (
	"log"
	"time"

	socketio "github.com/googollee/go-socket.io"
)

func main() {
	// Simple client to talk to default-http example
	uri := "http://127.0.0.1:8000"

	client, err := socketio.NewClient(uri, nil)
	if err != nil {
		panic(err)
	}

	// Handle an incoming event
	client.OnEvent("reply", func(s socketio.Conn, msg string) {
		log.Println("Receive Message /reply: ", "reply", msg)
	})

	err = client.Connect()
	if err != nil {
		panic(err)
	}

	client.Emit("notice", "hello")

	time.Sleep(1 * time.Second)
	err = client.Close()
	if err != nil {
		panic(err)
	}
}
