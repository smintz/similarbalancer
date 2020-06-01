package balancer

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/smintz/similarbalancer/client"
	"github.com/smintz/similarbalancer/webserver"
)

func NoError(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}

func setup() {
	var backends []*Backend

	for i := 1; i < 10; i++ {
		dir, _ := ioutil.TempDir("/tmp", "similar")
		url := fmt.Sprintf("localhost:2000%d", i)
		w := webserver.NewWebServer(dir, url, uint(i)*1)
		go w.Serve()
		backends = append(backends, NewBackend("http://"+url))
		log.Println("added new backend", w)
	}
	pool := &ServerPool{backends: backends}

	s := &Server{pool: pool}
	server := http.Server{Addr: "localhost:10000", Handler: http.HandlerFunc(s.Balancer)}
	go server.ListenAndServe()

}
func TestRunTest(t *testing.T) {
	c := client.Client{BaseURL: "http://localhost:10000"}
	NoError(t, c.Register("smintz", "pass"))
	NoError(t, c.Login("smintz"))
}

func TestMany(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			c := client.Client{BaseURL: "http://localhost:10000"}
			l, err := c.RegisterRandom()
			NoError(t, err)
			NoError(t, c.Login(l.Username))
			wg.Done()
		}()
	}
	wg.Wait()

}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}
