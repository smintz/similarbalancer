package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/smintz/similarbalancer/balancer"
	"github.com/smintz/similarbalancer/webserver"
)

type BackendsArray []*balancer.Backend

func (i *BackendsArray) String() string {
	arr := []string{}
	for _, s := range *i {
		arr = append(arr, s.URL.String())
	}
	return fmt.Sprintf("%v", arr)
}

func (i *BackendsArray) Set(value string) error {
	*i = append(*i, balancer.NewBackend(value))
	return nil
}

var (
	ADDR        = flag.String("addr", ":10000", "Listen address for load balancer")
	BACKENDS    = BackendsArray{}
	testServers = flag.Int("devservers", 0, "number of webservers to add")
)

func main() {
	flag.Var(&BACKENDS, "b", "backend address (can repeat)")
	flag.Parse()

	for i := 1; i < *testServers; i++ {
		dir, _ := ioutil.TempDir("/tmp", "similar")
		url := fmt.Sprintf("localhost:2000%d", i)
		w := webserver.NewWebServer(dir, url, uint(i)*1)
		go w.Serve()
		BACKENDS = append(BACKENDS, balancer.NewBackend("http://"+url))
		log.Println("added new backend", w)
	}
	log.Println("starting balancer at", *ADDR, BACKENDS.String())
	pool := balancer.NewServerPool(BACKENDS)

	s := balancer.NewServer(pool)
	server := http.Server{Addr: *ADDR, Handler: http.HandlerFunc(s.Balancer)}
	server.ListenAndServe()
}
