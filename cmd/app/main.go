package main

import (
	"context"

	"github.com/blinkinglight/bee"
	"github.com/nats-io/nats.go"
)

func main() {
	ctx := context.Background()
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		panic(err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		panic(err)
	}

	bee.WithNats(ctx, nc)
	bee.WithJetStream(ctx, js)

}
