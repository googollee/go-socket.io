// Package main runs a go-socket.io based websocket server with Iris web server.
package main

import (
	"log"

	"github.com/kataras/iris/v12"

	socketio "github.com/googollee/go-socket.io"
)

func main() {
	app := iris.New()

	server := socketio.NewServer(nil)

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		log.Println("connected:", s.ID())
		return nil
	})

	server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		log.Println("notice:", msg)
		s.Emit("reply", "have "+msg)
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
		log.Println("meet error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("closed", reason)
	})

	go server.Serve()
	defer server.Close()

	app.HandleMany("GET POST", "/socket.io/{any:path}", iris.FromStd(server))
	app.HandleDir("/", "../asset")

	if err := app.Run(
		iris.Addr(":8000"),
		iris.WithoutPathCorrection,
		iris.WithoutServerError(iris.ErrServerClosed),
	); err != nil {
		log.Fatal("failed run app: ", err)
	}
}
