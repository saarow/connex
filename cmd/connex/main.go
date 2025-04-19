package main

import "github.com/saarow/connex/pkg/tcp"

func main() {
	config := tcp.DefaultClientConfig(":9999")
	client := tcp.NewClient(config)
	client.Start()
	client.End()
}
