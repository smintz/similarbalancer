package main

import (
	"flag"
	"log"

	"github.com/smintz/similarbalancer/webserver"
)

var (
	DIR  = flag.String("dir", "/tmp/similarbalancer", "Where to store the user files")
	ADDR = flag.String("addr", ":8888", "Listen address")
	RATE = flag.Uint("rate", 0, "Target error rate")
)

func main() {
	flag.Parse()
	log.Println("starting webserver on", *ADDR)

	server := webserver.NewWebServer(*DIR, *ADDR, *RATE)
	server.Serve()

}
