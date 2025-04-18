package main

import (
	"github.com/saarow/connex/pkg/udp"
)

func main() {
	conf := udp.ClientConfig{
		RemoteAddress: "localhost:8080",
		LocalAddres:   ":9999",
	}

	client := udp.NewClient(conf)
	client.Start()
	client.End()
}
