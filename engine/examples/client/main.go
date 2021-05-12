package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/googollee/go-socket.io/engine"
)

func main() {
	ctx := context.Background()
	client, err := engine.NewClient(
		engine.OptionMaxBufferSize(1*1024*1024),
		engine.OptionLogLevel(engine.LogDebug),
		engine.OptionTransports("polling", "sse", "websocket"),
	)
	if err != nil {
		log.Fatal(err)
	}

	client.OnPacket(func(ctx engine.Context, type_ engine.PacketType, r io.Reader) {
		var data [1024]byte
		n, _ := r.Read(data[:])
		log.Printf("session %s get package %v with payload %s", ctx.Session().ID(), type_, string(data[n]))
	})

	client.OnMessage(func(ctx engine.Context, msg io.Reader) {
		var data [1024]byte
		n, err := msg.Read(data[:])
		if err != nil {
			log.Fatalf("read from engineio sid %s error: %s", ctx.Session().ID(), err)
			ctx.Session().Close()
			return
		}

		fmt.Println(string(data[:n]))
	})

	client.OnError(func(ctx engine.Context, err error) {
		log.Printf("engineio sid %s got error: %s", ctx.Session().ID(), err)
		ctx.Session().Close()
	})

	client.OnClosed(func(ctx engine.Context) {
		url := ctx.Session().Get("url").(string)
		log.Printf("engineio sid %s from %s closed", ctx.Session().ID(), url)
	})

	req, err := http.NewRequest("GET", "https://host/endpoint", nil)
	if err != nil {
		log.Fatal(err)
	}

	// client uses heads/url in the same req for following connections.
	if err := client.Dial(ctx, req); err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	w, err := client.SendFrame(engine.FrameText)
	if err != nil {
		log.Fatalln(err)
	}
	w.Write([]byte("hello engine.io!\n"))
	w.Close()

	time.Sleep(time.Second)
}
