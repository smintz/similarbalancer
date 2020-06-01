package main

import (
	"flag"
	"log"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/smintz/similarbalancer/client"
	"github.com/smintz/similarbalancer/structs"
)

var (
	postRPS   = flag.Int("post", 100, "Number of POST commands per second")
	getRPS    = flag.Int("get", 1000, "Number of GET commands per second")
	serverURL = flag.String("addr", "http://localhost:10000", "URL of server")
)

func main() {
	flag.Parse()
	log.Println("starting client", *postRPS)
	cache, err := lru.New(*postRPS)
	if err != nil {
		panic(err)
	}
	var wg sync.WaitGroup
	now := time.Now()
	for i := 0; i <= *postRPS; i++ {
		wg.Add(1)
		go func() {
			c := &client.Client{*serverURL}

			l, err := c.RegisterRandom()
			if err != nil {
				log.Println("error:", err)
				return
			}
			cache.Add(i, l)

			log.Println(l)
			wg.Done()
		}()

	}
	wg.Wait()
	log.Println("called post", *postRPS, "times in ", time.Since(now))
	var wgGet sync.WaitGroup
	now = time.Now()
	for i := 0; i <= *getRPS; i++ {
		wgGet.Add(1)
		go func() {
			c := &client.Client{*serverURL}
			log.Println(i % *postRPS)
			if k, l, ok := cache.GetOldest(); ok {
				lo := l.(*structs.LoginDetails)
				log.Println(k, lo)
				err = c.Login(lo.Username)
				if err != nil {
					log.Println("error:", err)
					return
				}

			}
			wgGet.Done()
		}()

	}
	wgGet.Wait()
	log.Println("called get", *getRPS, "times in ", time.Since(now))

}
