package main

import (
	"encoding/hex"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/googollee/go-engine.io"
)

func main() {
	server, err := engineio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			conn, _ := server.Accept()
			go func() {
				defer conn.Close()
				for {
					t, r, err := conn.NextReader()
					if err != nil {
						return
					}
					b, err := ioutil.ReadAll(r)
					if err != nil {
						return
					}
					r.Close()
					if t == engineio.MessageText {
						log.Println(t, string(b))
					} else {
						log.Println(t, hex.EncodeToString(b))
					}
					w, err := conn.NextWriter(t)
					if err != nil {
						return
					}
					w.Write([]byte("pong"))
					w.Close()
				}
			}()
		}
	}()

	http.Handle("/engine.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving at localhost:5000...")
	log.Fatal(http.ListenAndServe(":5000", nil))
}
