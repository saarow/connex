package main

import (
	"github.com/saarow/connex/pkg/tcp"
)

func main() {
	config := tcp.DefaultClientConfig("localhost:8080")
	client := tcp.NewClient(config)

	client.Start()
	client.End()
}
