package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/googollee/go-engine.io"
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

				time.Sleep(time.Second * 5)

				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer fmt.Println(conn.ID(), "write quit")

					w, err := conn.NextWriter(engineio.TEXT)
					if err != nil {
						log.Fatalln("next write error:", err)
						return
					}
					if _, err := w.Write([]byte("hello")); err != nil {
						log.Fatalln("write error:", err)
						return
					}
					if err := w.Close(); err != nil {
						log.Fatalln("write close error:", err)
						return
					}

					w, err = conn.NextWriter(engineio.BINARY)
					if err != nil {
						log.Fatalln("next write error:", err)
						return
					}
					if _, err := w.Write([]byte{1, 2, 3, 4}); err != nil {
						log.Fatalln("write error:", err)
						return
					}
					if err := w.Close(); err != nil {
						log.Fatalln("write close error:", err)
						return
					}
				}()

				typ, r, err := conn.NextReader()
				if err != nil {
					log.Fatalln("next read error:", err)
					return
				}
				b, err := ioutil.ReadAll(r)
				if err != nil {
					log.Fatalln("read all error:", err)
					return
				}
				fmt.Println("read:", typ, string(b))
				if string(b) != "hello" || typ != engineio.TEXT {
					log.Fatalln("read text error")
					return
				}
				if err := r.Close(); err != nil {
					log.Fatalln("close reader error:", err)
					return
				}

				typ, r, err = conn.NextReader()
				if err != nil {
					log.Fatalln("next read error:", err)
					return
				}
				b, err = ioutil.ReadAll(r)
				if err != nil {
					log.Fatalln("read all error:", err)
					return
				}
				fmt.Println("read:", typ, b)
				if b[0] != 1 || b[1] != 2 || b[2] != 3 || b[3] != 4 || typ != engineio.BINARY {
					log.Fatalln("read text error")
					return
				}
				if err := r.Close(); err != nil {
					log.Fatalln("close reader error:", err)
					return
				}

				wg.Wait()
				os.Exit(0)
			}(conn)
		}
	}()

	http.Handle("/engine.io/", server)
	fmt.Println("Serving at localhost:8080...")
	log.Fatal(http.ListenAndServe("localhost:8080", nil))
}
