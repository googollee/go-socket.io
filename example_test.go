package socketio

import (
	"fmt"
	"io"
	"log"
	"net/http/httptest"

	engineio "github.com/googollee/go-socket.io/connection"
	"github.com/googollee/go-socket.io/connection/base"
)

func ExampleServer() {
	eio, err := NewServer(nil)
	if err != nil {
		log.Fatalln("server error:", err)
	}
	httpSvr := httptest.NewServer(eio)
	defer httpSvr.Close()

	for {
		conn, err := eio.Accept()
		if err != nil {
			log.Fatalln("accept error:", err)
			return
		}
		go func(conn engineio.Conn) {
			defer conn.Close()
			fmt.Println(conn.ID(), conn.RemoteAddr(), "->", conn.LocalAddr(), "with", conn.RemoteHeader())

			for {
				typ, _, r, err := conn.NextReader()
				if err != nil {
					log.Fatalln("read error:", err)
					return
				}
				w, err := conn.NextWriter(typ, base.MESSAGE)
				if err != nil {
					r.Close()
					log.Fatalln("write error:", err)
					return
				}
				_, err = io.Copy(w, r)
				if err != nil {
					r.Close()
					w.Close()
					log.Fatalln("copy error:", err)
					return
				}
				if err = w.Close(); err != nil {
					log.Fatalln("close writer error:", err)
					return
				}
				if err = r.Close(); err != nil {
					log.Fatalln("close reader error:", err)
					return
				}
			}
		}(conn)
	}
}
