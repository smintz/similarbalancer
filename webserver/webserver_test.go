package webserver

import (
	"io/ioutil"
	"testing"

	"github.com/smintz/similarbalancer/client"
)

func NoError(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}

func TestRunTest(t *testing.T) {
	dir, _ := ioutil.TempDir("/tmp", "similarbalancer")
	server := &WebServer{basePath: dir, listenAddress: ":8888"}
	go server.Serve()

	c := &client.Client{BaseURL: "http://localhost:8888"}
	NoError(t, c.Register("smintz1", "secretpass"))
	NoError(t, c.Login("smintz1"))
}
