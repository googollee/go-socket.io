package main

import (
	"log"

	"github.com/gogf/gf/frame/g"
	"github.com/gogf/gf/net/ghttp"

	socketio "github.com/googollee/go-socket.io"
)

func cors(r *ghttp.Request) {
	r.Response.CORSDefault()
	r.Middleware.Next()
}

func main() {
	s := g.Server()

	server := socketio.NewServer(nil)

	s.BindMiddlewareDefault(cors)
	s.BindHandler("/socket.io/", func(r *ghttp.Request) {
		server.ServeHTTP(r.Response.Writer, r.Request)
	})

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

	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()
	defer server.Close()

	s.SetPort(8000)
	s.Run()
}
