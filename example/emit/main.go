package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
	"sync"

	socketio "github.com/googollee/go-socket.io"
)

var (
	connections sync.Map
)

func init() {
	connections = sync.Map{}
}

func main() {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected:", s.ID())
		connections.Store(s.ID(), true)
		return nil
	})
	server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		fmt.Println("notice:", msg)
		server.Emit(s.ID(), "reply", "notice message " + msg)
	})
	server.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
		s.SetContext(msg)
		server.Emit(s.ID(), "reply", "chat msg " + msg)
		fmt.Println("connect id", s.ID())
		return "recv " + msg
	})
	server.OnEvent("/", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		s.Emit("bye", last)
		s.Close()
		return last
	})
	server.OnError("/", func(e error) {
		fmt.Println("meet error:", e)
	})
	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		fmt.Println("closed", msg)
	})
	go server.Serve()
	defer server.Close()

	timer := time.NewTimer(time.Second * 10)
	go func() {
		<-timer.C
		testId := "1"
		if _, ok := connections.Load(testId); ok {
			server.Emit(testId, "reply", "wow! I can emit message by connetion Id")
		}
	}()

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("../asset")))
	log.Println("Serving at localhost:8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
