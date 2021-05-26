package main

import (
	"log"
	"net/http"
	"time"

	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/transport"
)

func main() {
	eio, err := engineio.New(
		engineio.OptionTransports(transport.Polling, transport.SSE, transport.Websocket),
		engineio.OptionJSONP(4), // polling with jsonp `__eio[4]("packet data")`
		engineio.OptionPingInterval(25*time.Second),
		engineio.OptionPingTimeout(50*time.Second),
		engineio.OptionMaxBufferSize(1*1024*1024), // 1 MiB
		engineio.OptionLogger(nil),
	)
	if err != nil {
		log.Fatalln(err)
	}

	eio.With(func(ctx *engineio.Context) {
		log.Printf("session %s get %v packet", ctx.Session.ID(), ctx.Packet.Type)
		ctx.Next()
	})

	eio.OnOpen(func(ctx *engineio.Context) error {
		log.Printf("engineio sid %s opened with transport %s", ctx.Session.ID(), ctx.Session.Transport())
		ctx.Session.Store("url", ctx.Request.URL.String())
		if ctx.Request.URL.Query().Get("allow") == "false" {
			// An error means server won't continue this session and return a non-2xx response.
			return engineio.HTTPError(http.StatusNotAcceptable, "client says allow == false")
		}

		return nil
	})

	eio.OnMessage(func(ctx *engineio.Context) {
		var data [1024]byte
		n, err := ctx.Packet.Body.Read(data[:])
		if err != nil {
			log.Fatalf("read from engineio sid %s error: %s", ctx.Session.ID(), err)
			ctx.Session.Close()
			return
		}

		writer, err := ctx.Session.SendFrame(frame.Text)
		if err != nil {
			log.Fatalf("next writer from engineio sid %s error: %s", ctx.Session.ID(), err)
			ctx.Session.Close()
			return
		}
		defer writer.Close()

		if _, err := writer.Write(data[:n]); err != nil {
			log.Fatalf("write to engineio sid %s error: %s", ctx.Session.ID(), err)
			ctx.Session.Close()
			return
		}
	})

	eio.OnError(func(ctx *engineio.Context, err error) {
		log.Printf("engineio sid %s got error: %s", ctx.Session.ID(), err)
		ctx.Session.Close()
	})

	eio.OnClose(func(ctx *engineio.Context) {
		url := ctx.Session.Get("url").(string)
		log.Printf("engineio sid %s from %s closed", ctx.Session.ID(), url)
	})

	// engineio.Server implements http.Handler, which compatibles with any http frameworks.
	// Make sure routing all methods of an endpoint to engineio.Server.
	http.Handle("/engineio", eio)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
