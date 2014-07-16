package main

import (
	".."
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	server := engineio.NewServer(engineio.DefaultConfig)

	go func() {
		for {
			conn, _ := server.Accept()
			go func() {
				defer conn.Close()
				for i := 0; i < 10; i++ {
					r, _ := conn.NextReader()
					b, _ := ioutil.ReadAll(r)
					r.Close()
					log.Println(string(b))
					w, _ := conn.NextWriter(engineio.MessageText)
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
