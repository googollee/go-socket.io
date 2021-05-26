package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/frame"
	"github.com/googollee/go-socket.io/engineio/transport"
)

func main() {
	ctx := context.Background()
	client, err := engineio.NewClient(
		engineio.OptionMaxBufferSize(1*1024*1024),
		engineio.OptionTransports(transport.Polling, transport.Websocket, transport.SSE),
	)
	if err != nil {
		log.Fatal(err)
	}

	client.With(func(ctx *engineio.Context) {
		log.Printf("session %s get %v packet", ctx.Session.ID(), ctx.Packet.Type)
		ctx.Next()
	})

	client.OnMessage(func(ctx *engineio.Context) {
		var data [1024]byte
		n, err := ctx.Packet.Body.Read(data[:])
		if err != nil {
			log.Fatalf("read from engineio sid %s error: %s", ctx.Session.ID(), err)
			ctx.Session.Close()
			return
		}

		fmt.Println(string(data[:n]))
	})

	client.OnError(func(ctx *engineio.Context, err error) {
		log.Printf("engineio sid %s got error: %s", ctx.Session.ID(), err)
		ctx.Session.Close()
	})

	client.OnClose(func(ctx *engineio.Context) {
		url := ctx.Session.Get("url").(string)
		log.Printf("engineio sid %s from %s closed", ctx.Session.ID(), url)
	})

	req, err := http.NewRequestWithContext(ctx, "GET", "https://host/endpoint", nil)
	if err != nil {
		log.Fatal(err)
	}

	// client uses heads/url in the same req for following connections.
	if err := client.Open(req); err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	w, err := client.SendFrame(frame.Text)
	if err != nil {
		log.Fatalln(err)
	}
	w.Write([]byte("hello engineio.io!\n"))
	w.Close()

	time.Sleep(time.Second)
}
