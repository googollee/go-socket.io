// Simple Socket.io Proxy
// call go run proxy.go <remote address>

package main

import (
	"log"
	"net/http"
	"os"

	socketio "github.com/googollee/go-socket.io"
)

func main() {
	remote := os.Args[1]
	log.Println("PROXYING REMOTE", remote)
	server := socketio.NewServer(nil)

	server.OnConnect("/", func(s socketio.Conn) error {
		log.Println("SERVER OnConnect")
		return nil
	})

	server.OnError("/", func(s socketio.Conn, err error) {
		log.Println("SERVER OnError", err)
	})

	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		log.Println("SERVER OnDisconnect", msg)
	})

	server.OnEvent("/", "*", func(s socketio.Conn, e socketio.EventRequest, msg interface{}) interface{} {
		log.Println("SERVER RECIEVED EVENT", e.Event(), "WITH MESSAGE", msg)
		// Forwarding messages from the caller to the server
		if msg == nil {
			return nil
		}
		client := findOrCreateClient(remote, s.ID(), s)

		messageReturn := make(chan interface{})
		go func() {
			client.Emit(e.Event(), msg, func(ar interface{}) {
				messageReturn <- ar
			})
		}()
		ar := <-messageReturn

		log.Println("SERVER RETURNING EVENT", e.Event(), "WITH MESSAGE", msg, "VALUE", ar)
		return ar
	})

	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()
	defer server.Close()

	http.Handle("/socket.io/", server)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

var idToClient map[string]*socketio.Client = map[string]*socketio.Client{}

func findOrCreateClient(remote string, id string, serverSock socketio.Conn) *socketio.Client {
	client, ok := idToClient[id]
	if ok {
		log.Println("Found client in cache", id)
		return client
	}

	client, err := socketio.NewClient(remote, nil)
	if err != nil {
		panic(err)
	}

	client.OnError(func(s socketio.Conn, err error) {
		log.Println("CLIENT OnError", err)
	})

	client.OnConnect(func(s socketio.Conn) error {
		// Called on every successful Get Request, so pretty useless really
		log.Println("CLIENT OnConnect")
		return nil
	})

	client.OnDisconnect(func(s socketio.Conn, msg string) {
		// Called on every successful Get Request, so pretty useless really
		log.Println("CLIENT OnDisconnect", msg)
	})

	client.OnEvent("*", func(s socketio.Conn, e socketio.EventRequest, msg interface{}) {
		// Messages coming back from the server to the caller
		log.Println("CLIENT RECIEVED EVENT", e.Event(), "WITH MESSAGE", msg)
		if msg == nil {
			serverSock.Emit(e.Event())
		} else {
			serverSock.Emit(e.Event(), msg)
		}
	})

	log.Println("CLIENT CONNECTING")
	client.Connect()
	idToClient[id] = client

	return client
}
