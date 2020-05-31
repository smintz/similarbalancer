package webserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/smintz/similarbalancer/structs"
)

type WebServer struct {
	basePath      string
	listenAddress string
}

func (s *WebServer) Register(w http.ResponseWriter, req *http.Request) {
	var u structs.LoginDetails

	err := json.NewDecoder(req.Body).Decode(&u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Println("Register request with", u)

	jsonBytes, _ := json.Marshal(u)
	err = ioutil.WriteFile(filepath.Join(s.basePath, u.Username), jsonBytes, os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Operation for %s succeeded", u.Username)

}

func (s *WebServer) Login(w http.ResponseWriter, req *http.Request) {
	var u structs.LoginDetails
	username := req.Response.Request.URL.Query().Get("username")

	log.Println("Requesting details for", username)
	file, err := os.Open(filepath.Join(s.basePath, username))
	defer file.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	err = json.NewDecoder(file).Decode(&u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Read for %s succeeded", u.Username)
}

func (s *WebServer) Serve() {
	http.HandleFunc("/register", s.Register)
	http.HandleFunc("/changePassword", s.Register)
	http.HandleFunc("/login", s.Login)

	http.ListenAndServe(s.listenAddress, nil)
}
