package webserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/smintz/similarbalancer/structs"
)

type WebServer struct {
	basePath      string
	listenAddress string
	ErrorRate     uint
}

func NewWebServer(path, listenAddress string, errorRate uint) *WebServer {
	if errorRate > 100 {
		panic("error rate must be under 100")
	}
	return &WebServer{
		basePath:      path,
		listenAddress: listenAddress,
		ErrorRate:     errorRate,
	}
}

func (s *WebServer) Register(w http.ResponseWriter, req *http.Request) {
	var u structs.LoginDetails

	rand.Seed(time.Now().UnixNano())
	rs := rand.Int() % 100
	if int(s.ErrorRate) > rs {
		http.Error(w, "failed randomly", http.StatusInternalServerError)
		log.Printf("failed randomly (%v/%v)", s.ErrorRate, rs)
		return
	}

	err := json.NewDecoder(req.Body).Decode(&u)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Println(s.listenAddress, "Register request with", u)

	jsonBytes, _ := json.Marshal(u)
	err = ioutil.WriteFile(filepath.Join(s.basePath, u.Username), jsonBytes, os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(201)
	fmt.Fprintf(w, "Operation for %s succeeded", u.Username)

}

func (s *WebServer) Login(w http.ResponseWriter, req *http.Request) {
	var u structs.LoginDetails
	username := req.URL.Query().Get("username")

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
	log.Println("registering for", s)
	mux := http.NewServeMux()
	mux.HandleFunc("/register", s.Register)
	mux.HandleFunc("/changePassword", s.Register)
	mux.HandleFunc("/login", s.Login)

	http.ListenAndServe(s.listenAddress, mux)
}
