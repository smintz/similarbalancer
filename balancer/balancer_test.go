package balancer

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"github.com/smintz/similarbalancer/client"
	"github.com/smintz/similarbalancer/webserver"
)

func NoError(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}
func TestRunTest(t *testing.T) {

	var backends []*Backend

	for i := 0; i < 10; i++ {
		dir, _ := ioutil.TempDir("/tmp", "similar")
		url := fmt.Sprintf("localhost:2000%d", i)
		w := webserver.NewWebServer(dir, url, uint(i)*5)
		go w.Serve()
		backends = append(backends, NewBackend("http://"+url))
		log.Println("added new backend", w)
	}
	pool := &ServerPool{backends: backends}

	s := &Server{pool: pool}
	server := http.Server{Addr: "localhost:10000", Handler: http.HandlerFunc(s.lb)}
	go server.ListenAndServe()

	client := client.Client{BaseURL: "http://localhost:10000"}
	NoError(t, client.Register("smintz", "pass"))
	NoError(t, client.Login("smintz"))

}
