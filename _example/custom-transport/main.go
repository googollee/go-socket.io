package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	engineio "github.com/googollee/go-engine.io"
	"github.com/googollee/go-engine.io/transport"
	"github.com/googollee/go-engine.io/transport/polling"
	"github.com/googollee/go-engine.io/transport/websocket"
	socketio "github.com/googollee/go-socket.io"
)

func main() {
	v := url.Values{
		"username": []string{"test_username"},
	}
	myUrl := url.URL{
		RawQuery: v.Encode(),
	}
	pollTransport := polling.Default
	wsTransport := websocket.Default

	pollTransport.SetURL(myUrl)
	pollTransport.SetURL(myUrl)

	options := engineio.Options{
		Transports: []transport.Transport{
			pollTransport,
			wsTransport,
		},
	}

	server, err := socketio.NewServer(&options)
	if err != nil {
		log.Fatal(err)
	}
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected:", s.ID())
		return nil
	})
	server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		fmt.Println("notice:", msg)
		server.Emit(s.ID(), "reply", "notice message "+msg)
	})
	server.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
		s.SetContext(msg)
		server.Emit(s.ID(), "reply", "chat msg "+msg)
		fmt.Println("connect id", s.ID())
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
	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		fmt.Println("closed", msg)
	})
	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("../asset")))
	log.Println("Serving at localhost:8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
