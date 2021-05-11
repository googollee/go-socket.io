package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/googollee/go-socket.io/engine"
)

func main() {
	eio, err := engine.New(
		engine.OptionTransports("polling", "sse", "websocket"),
		engine.OptionPingInterval(25*time.Second),
		engine.OptionPingTimeout(50*time.Second),
		engine.OptionMaxBufferSize(1*1024*1024), // 1 MiB
		engine.OptionLogLevel(engine.LogDebug),
		engine.OptionLogger(nil),
	)
	if err != nil {
		log.Fatalln(err)
	}

	eio.OnOpen(func(ctx engine.Context, req *http.Request) error {
		log.Printf("engineio sid %s opened with transport %s", ctx.Session().ID(), ctx.Session().Transport())
		ctx.Session().Store("url", req.URL.String())
		if req.URL.Query().Get("allow") == "false" {
			// An error means server won't continue this session and return a non-2xx response.
			return engine.HTTPError(http.StatusNotAcceptable, "client says allow == false")
		}

		return nil
	})

	eio.OnPingPong(func(ctx engine.Context) {
		log.Printf("engineio sid %s got ping", ctx.Session().ID())
	})

	eio.OnUpgrade(func(ctx engine.Context, req *http.Request) error {
		log.Printf("engineio sid %s upgraded to transport %s", ctx.Session().ID(), ctx.Session().Transport())
		return nil
	})

	eio.OnMessage(func(ctx engine.Context, msg io.Reader) {
		var data [1024]byte
		n, err := msg.Read(data[:])
		if err != nil {
			log.Fatalf("read from engineio sid %s error: %s", ctx.Session().ID(), err)
			ctx.Session().Close()
			return
		}

		writer, err := ctx.Session().SendFrame(engine.FrameText)
		if err != nil {
			log.Fatalf("next writer from engineio sid %s error: %s", ctx.Session().ID(), err)
			ctx.Session().Close()
			return
		}
		defer writer.Close()

		if _, err := writer.Write(data[:n]); err != nil {
			log.Fatalf("write to engineio sid %s error: %s", ctx.Session().ID(), err)
			ctx.Session().Close()
			return
		}
	})

	eio.OnError(func(ctx engine.Context, err error) {
		log.Printf("engineio sid %s got error: %s", ctx.Session().ID(), err)
		ctx.Session().Close()
	})

	eio.OnClosed(func(ctx engine.Context) {
		url := ctx.Session().Get("url").(string)
		log.Printf("engineio sid %s from %s closed", ctx.Session().ID(), url)
	})

	// engine.Server implements http.Handler, which compatibles with any http frameworks.
	// Make sure routing all methods of an endpoint to engine.Server.
	http.Handle("/engineio", eio)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
