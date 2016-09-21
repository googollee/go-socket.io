package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/googollee/go-engine.io"
	"github.com/googollee/go-engine.io/transport"
	"github.com/googollee/go-engine.io/transport/polling"
)

func main() {
	dialer := engineio.Dialer{
		Transports: []transport.Transport{polling.Default},
	}
	conn, err := dialer.Dial("http://localhost:8080/engine.io/", nil)
	if err != nil {
		log.Fatalln("dial error:", err)
	}
	defer conn.Close()
	fmt.Println(conn.ID(), conn.LocalAddr(), "->", conn.RemoteAddr(), "with", conn.RemoteHeader())

	go func() {
		defer conn.Close()

		ft, r, err := conn.NextReader()
		if err != nil {
			log.Println("next reader error:", err)
			return
		}
		b, err := ioutil.ReadAll(r)
		if err != nil {
			log.Println("read all error:", err)
			return
		}
		if err := r.Close(); err != nil {
			log.Println("read close:", err)
			return
		}
		fmt.Println("read:", ft, b)
	}()

	for {
		w, err := conn.NextWriter(engineio.TEXT)
		if err != nil {
			log.Println("next writer error:", err)
			return
		}
		if _, err := w.Write([]byte("hello")); err != nil {
			log.Println("write error:", err)
			return
		}
		if err := w.Close(); err != nil {
			log.Println("write close error:", err)
			return
		}
		w, err = conn.NextWriter(engineio.BINARY)
		if err != nil {
			log.Println("next writer error:", err)
			return
		}
		if _, err := w.Write([]byte{1, 2, 3, 4}); err != nil {
			log.Println("write error:", err)
			return
		}
		if err := w.Close(); err != nil {
			log.Println("write close error:", err)
			return
		}
		time.Sleep(time.Second * 5)
	}
}
