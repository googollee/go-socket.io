package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	engineio "github.com/googollee/go-socket.io/connection"
)

func main() {
	server, err := engineio.NewServer(nil)
	if err != nil {
		log.Fatal("server error:", err)
	}

	go func() {
		for {
			conn, err := server.Accept()
			if err != nil {
				log.Fatalln("accept error:", err)
				return
			}

			go func(conn engineio.Conn) {
				defer conn.Close()
				fmt.Println(conn.ID(), conn.RemoteAddr(), "->", conn.LocalAddr(), "with", conn.RemoteHeader())

				for {
					ft, r, err := conn.NextReader()
					if err != nil {
						fmt.Println("read error:", err)
						return
					}
					fmt.Println("read type:", ft)

					w, err := conn.NextWriter(ft)
					if err != nil {
						r.Close()
						fmt.Println("write error:", err)
						return
					}

					_, err = io.Copy(w, r)
					w.Close()
					r.Close()
					if err != nil {
						fmt.Println("copy error:", err)
						return
					}
				}
			}(conn)
		}
	}()

	http.Handle("/engine.io/", server)
	dir := http.Dir("./asset")
	f, err := dir.Open("index.html")
	if err != nil {
		fmt.Println("need run under go-engine.io /demo/web directory.")
		return
	}
	f.Close()
	http.Handle("/", http.FileServer(dir))
	fmt.Println("Serving at localhost:5000...")
	log.Fatal(http.ListenAndServe("localhost:5000", nil))
}
