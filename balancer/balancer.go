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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

func NewServerPool(b []*Backend) *ServerPool {
	return &ServerPool{
		backends: b,
		current:  0,
	}
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

type NopResponseWriter struct {
	status int
	mux    sync.RWMutex
}

func NewNopResponseWriter() *NopResponseWriter {
	return &NopResponseWriter{status: 0}
}

func (w *NopResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}
func (w *NopResponseWriter) WriteHeader(code int) {
	w.mux.Lock()
	w.status = code
	w.mux.Unlock()
}

func (w *NopResponseWriter) Header() http.Header {
	return http.Header{}
}

func (w *NopResponseWriter) Status() int {
	w.mux.RLock()
	defer w.mux.RUnlock()
	return w.status
}

var retriesCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "balancer_retries_count",
	Help: "A Counter to count the number of retries to each backend",
}, []string{"method", "path", "backend"})

func (s *ServerPool) Broadcast(w http.ResponseWriter, r *http.Request) error {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	log.Printf("request: %v", b)
	var wg sync.WaitGroup
	for _, peer := range s.backends {
		wg.Add(1)
		tries := 0
		go func(p *Backend) {
			retry.Do(func() error {
				// ctx, _ := context.WithTimeout(r.Context(), 5*time.Second)
				ctx := context.TODO()
				rr := r.Clone(ctx)
				rr.Body = ioutil.NopCloser(bytes.NewReader(b))
				ww := NewNopResponseWriter()

				tries++
				log.Println("trying", p.URL, "tries:", tries)
				p.ReverseProxy.ServeHTTP(ww, rr)
				if ww.Status() != 201 {
					retriesCount.WithLabelValues(rr.Method, rr.URL.Path, p.URL.String()).Inc()
					log.Println(p.URL, "failed with status", ww.Status())
					return fmt.Errorf("Status is %v", ww.Status())
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

func NewServer(p *ServerPool) *Server {
	return &Server{
		pool: p,
	}
}

var histogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "balancer_requests_seconds",
	Help: "An hitogram observing time to server requests by backends",
}, []string{"method", "path", "backend"})

// lb load balances the incoming request
func (s *Server) Balancer(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	r.Body = ioutil.NopCloser(bytes.NewReader(b))

	peer := s.pool.GetNextPeer()
	if peer != nil {
		timer := prometheus.NewTimer(histogram.WithLabelValues(r.Method, r.URL.Path, peer.URL.String()))
		peer.ReverseProxy.ServeHTTP(w, r)
		timer.ObserveDuration()
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
	if r.Method == "POST" {
		log.Println("got post message")
		rr := r.Clone(context.TODO())
		rr.Body = ioutil.NopCloser(bytes.NewReader(b))

		s.pool.Broadcast(w, rr)
	}
}

//   server := http.Server{
// 	Addr:    fmt.Sprintf(":%d", port),
// 	Handler: http.HandlerFunc(lb),
//   }
