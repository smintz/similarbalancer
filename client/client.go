package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/smintz/similarbalancer/structs"
)

type Client struct {
	BaseURL string
}

func (c *Client) Register(username, password string) error {
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(&structs.LoginDetails{Username: username, Password: password})
	_, err := http.Post(c.BaseURL+"/register", "application/json", buf)
	return err
}

func (c *Client) ChangePassword(username, password string) error {
	return c.Register(username, password)
}

func (c *Client) Login(username string) error {
	r, err := http.Get(c.BaseURL + "/login?username=" + username)
	if err != nil {
		return err
	}
	log.Println(r)
	if r.StatusCode > 299 {
		return fmt.Errorf("Status code is %v (%v)", r.StatusCode, r.Status)
	}

	return err
}
