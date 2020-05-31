package client

import (
	"bytes"
	"encoding/json"
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
	_, err := http.Get(c.BaseURL + "/login?username=" + username)
	return err
}
