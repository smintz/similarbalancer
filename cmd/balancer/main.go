package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/smintz/similarbalancer/balancer"
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
	ADDR     = flag.String("addr", ":10000", "Listen address for load balancer")
	BACKENDS = BackendsArray{}
)

func main() {
	flag.Var(&BACKENDS, "b", "backend address (can repeat)")
	flag.Parse()
	log.Println("starting balancer at", *ADDR, BACKENDS.String())
	pool := balancer.NewServerPool(BACKENDS)

	s := balancer.NewServer(pool)
	server := http.Server{Addr: *ADDR, Handler: http.HandlerFunc(s.Balancer)}
	server.ListenAndServe()
}
