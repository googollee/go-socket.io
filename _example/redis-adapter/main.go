package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	socketio "github.com/googollee/go-socket.io"
)

func main() {
	router := gin.New()

	server := socketio.NewServer(nil)
	ok, err := server.Adapter(&socketio.RedisAdapterOptions{
		Host:   "127.0.0.1",
		Port:   "6379",
		Prefix: "socket.io",
	})

	if err != nil {
		fmt.Println("error:", err)
		return
	}

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		s.Join("chatroom")
		fmt.Println("connected:", s.ID())
		return nil
	})

	server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		fmt.Println("notice:", msg)
		server.BroadcastToRoom("/", "chatroom", msg)
	})

	server.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
		s.SetContext(msg)
		return "recv " + msg
	})

	server.OnEvent("/", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		s.Emit("bye", last)
		s.Close()
		return last
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		fmt.Println("meet error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Println("closed", reason)
	})

	go server.Serve()
	defer server.Close()

	router.GET("/socket.io/*any", gin.WrapH(server))
	router.POST("/socket.io/*any", gin.WrapH(server))
	router.StaticFS("/public", http.Dir("../asset"))

	router.Run()
}
