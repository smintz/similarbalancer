package balancer

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/avast/retry-go"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

func NewBackend(urlStr string) *Backend {
	u, _ := url.Parse(urlStr)
	rp := httputil.NewSingleHostReverseProxy(u)
	return &Backend{
		URL:          u,
		Alive:        true,
		ReverseProxy: rp,
	}
}

// SetAlive for this backend
func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

// IsAlive returns true when backend is alive
func (b *Backend) IsAlive() (alive bool) {
	b.mux.RLock()
	alive = b.Alive
	b.mux.RUnlock()
	return
}

type ServerPool struct {
	backends []*Backend
	current  uint64
}

func (s *ServerPool) Append(b *Backend) {
	s.backends = append(s.backends, b)
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

// GetNextPeer returns next active peer to take a connection
func (s *ServerPool) GetNextPeer() *Backend {
	// loop entire backends to find out an Alive backend
	next := s.NextIndex()
	l := len(s.backends) + next // start from next and move a full cycle
	for i := next; i < l; i++ {
		idx := i % len(s.backends) // take an index by modding with length
		// if we have an alive backend, use it and store if its not the original one
		if s.backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx)) // mark the current one
			}
			return s.backends[idx]
		}
	}
	return nil
}

type WrappedResponseWriter struct {
	sent   bool
	status int
	writer http.ResponseWriter
}

func (w *WrappedResponseWriter) Write(b []byte) (int, error) {
	log.Println("got response")
	if w.sent {
		log.Println("response alredy sent")
		return len(b), nil
	}
	return w.writer.Write(b)
}

func (w *WrappedResponseWriter) Header() http.Header {
	return w.writer.Header()
}

func (w *WrappedResponseWriter) WriteHeader(statusCode int) {
	log.Println("writing header", statusCode)
	w.writer.WriteHeader(statusCode)
	w.status = statusCode
	w.sent = true
}

func (s *ServerPool) Broadcast(w http.ResponseWriter, r *http.Request) error {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	log.Printf("request: %v", b)
	var wg sync.WaitGroup
	for _, peer := range s.backends {
		wg.Add(1)
		go func(p *Backend) {
			retry.Do(func() error {
				// ctx, _ := context.WithTimeout(r.Context(), 5*time.Second)
				ctx := context.TODO()
				rr := r.Clone(ctx)
				rr.Body = ioutil.NopCloser(bytes.NewReader(b))
				ww := &WrappedResponseWriter{sent: false, writer: w}

				p.ReverseProxy.ServeHTTP(ww, rr)
				if ww.status > 210 {
					log.Println(p.URL, "failed with status", ww.status)
					return fmt.Errorf("Status is %v", ww.status)
				}

				return nil
			})
			log.Println(p.URL, "done")
			wg.Done()
		}(peer)
	}
	wg.Wait()
	return nil
}

type Server struct {
	pool *ServerPool
}

// lb load balances the incoming request
func (s *Server) lb(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		peer := s.pool.GetNextPeer()
		if peer != nil {
			peer.ReverseProxy.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
	case "POST":
		s.pool.Broadcast(w, r)
	}
}

//   server := http.Server{
// 	Addr:    fmt.Sprintf(":%d", port),
// 	Handler: http.HandlerFunc(lb),
//   }
