package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/googollee/go-socket.io/engine"
)

func main() {
	eio := engine.New(engine.DefaultOption().
		WithPingInterval(25 * time.Second).
		WithPingTimeout(50 * time.Second))

	eio.OnOpen(func(ctx engine.Context, req *http.Request) error {
		log.Printf("engineio sid %s opened with transport %s", ctx.Session().ID(), ctx.Session().Transport())
		ctx.Session().Store("url", req.URL.String())
		if req.URL.Query().Get("allow") == "false" {
			// An error means server won't continue this session and return a non-2xx response.
			return engine.HTTPError(http.StatusNotAcceptable, "client says allow == false")
		}

		return nil
	})

	eio.OnPing(func(ctx engine.Context) {
		log.Printf("engineio sid %s got ping", ctx.Session().ID())
	})

	eio.OnUpgrade(func(ctx engine.Context, req *http.Request) error {
		log.Printf("engineio sid %s upgraded to transport %s", ctx.Session().ID(), ctx.Session().Transport())
		return nil
	})

	eio.OnMessage(func(ctx engine.Context, msg io.Reader) {
		data, err := io.ReadAll(msg)
		if err != nil {
			log.Fatalf("read from engineio sid %s error: %s", ctx.Session().ID(), err)
			ctx.Session().Close()
			return
		}

		writer, err := ctx.SendFrame(engine.FrameText)
		if err != nil {
			log.Fatalf("next writer from engineio sid %s error: %s", ctx.Session().ID(), err)
			ctx.Session().Close()
			return
		}
		defer writer.Close()

		if _, err := writer.Write(data); err != nil {
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
		log.Printf("engineio sid %s closed", ctx.Session().ID())
	})

	// engine.Server implements http.Handler, which compatibles with any http frameworks.
	// Make sure routing all methods of an endpoint to engine.Server.
	http.Handle("/engineio", eio)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
