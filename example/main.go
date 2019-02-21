package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	socketio "github.com/googollee/go-socket.io"
)

func main() {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected:", s.ID())
		i := 0
		go func() {
			fmt.Println("Starting mass messaging goroutine")
			// this will send lots of message to any connecting clients, problems :
			// 1 - client receive almost none of these messages (they are not even sent to the websocket)
			// 2 - sockets with clients tends to disconnect automatically
			for {
				// Emmit will block the goroutine if the socket is not responding, is that intended behaviour?
				mess := "some message :  " + strconv.Itoa(i)
				fmt.Println("Emitting : " + mess)
				s.Emit("reply", mess)
				i++
				time.Sleep(100 * time.Millisecond)
			}
		}()
		return nil
	})
	server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		fmt.Println("notice:", msg)
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
	server.OnError("/", func(e error) {
		fmt.Println("meet error:", e)
	})
	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		fmt.Println("closed", msg)
	})
	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving at localhost:8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
