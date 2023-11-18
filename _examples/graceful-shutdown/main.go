package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	socketio "github.com/googollee/go-socket.io"
)

const (
	port = ":8000"

	gracefulDelay = 3 * time.Second
)

func main() {
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

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("../asset")))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()

	go func() {
		if err := http.ListenAndServe(port, nil); err != nil {
			log.Fatalf("http listen error: %s\n", err)
		}
	}()

	log.Printf("server started by %v", port)

	<-done

	//shutdown delay
	log.Printf("graceful delay: %v\n", gracefulDelay)

	time.Sleep(gracefulDelay)

	log.Println("server stopped")

	if err := server.Close(); err != nil {
		log.Fatalf("server shutdown failed: %s\n", err)
	}

	log.Println("server is shutdown")
}
