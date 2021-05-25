package main

import (
	"log"
	"net/http"
	"net/http/pprof"
	_ "net/http/pprof"

	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"
)

var allowOriginFunc = func(r *http.Request) bool {
	return true
}

func main() {
	opts := &engineio.Options{
		Transports: []transport.Transport{
			&polling.Transport{
				CheckOrigin: allowOriginFunc,
			},
			&websocket.Transport{
				CheckOrigin: allowOriginFunc,
			},
		},
	}

	server := socketio.NewServer(opts)

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

	debugMux := http.NewServeMux()

	debugMux.HandleFunc("/pprof/*", pprof.Index)
	debugMux.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	debugMux.HandleFunc("/pprof/profile", pprof.Profile)
	debugMux.HandleFunc("/pprof/symbol", pprof.Symbol)
	debugMux.HandleFunc("/pprof/trace", pprof.Trace)

	go func() {
		log.Println("Serving debug at :8001...")

		if err := http.ListenAndServe(":8001", debugMux); err != nil {
			log.Fatalf("debug serve error: %s\n", err)
		}
	}()

	go func() {
		log.Println("Serving socketio...")

		if err := server.Serve(); err != nil {
			log.Fatalf("socketio serve error: %s\n", err)
		}
	}()
	defer server.Close()

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("../asset")))

	log.Println("Serving at :8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
