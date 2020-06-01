# Similar Balancer

- [webserver]: simple web server which writes to disk on POST and reads from disk on GET, can be set with artifactual error rate.
- [client]: simple client to assist with tests
- [balancer]: implements Round-Robin for backends, Broadcast to all backends when a POST is done.

### Running tests

    $ go test ./...

### Running manually

#### Option 1

    $ ulimit -Sn 65334 # must increase the NOFILE limit
    $ export SERVER=1; mkdir /tmp/similar${SERVER}; go run ./cmd/webserver/main.go -addr localhost:2000${SERVER}; rm -rvf /tmp/similar${SERVER}
    $ export SERVER=2; mkdir /tmp/similar${SERVER}; go run ./cmd/webserver/main.go -addr localhost:2000${SERVER}; rm -rvf /tmp/similar${SERVER}
    $ export SERVER=3; mkdir /tmp/similar${SERVER}; go run ./cmd/webserver/main.go -addr localhost:2000${SERVER}; rm -rvf /tmp/similar${SERVER}
    $ export SERVER=3; mkdir /tmp/similar${SERVER}; go run ./cmd/webserver/main.go -addr localhost:2000${SERVER}; rm -rvf /tmp/similar${SERVER}
    $ go run ./cmd/balancer/main.go -b http://localhost:20001 -b http://localhost:20002 -b http://localhost:20004 -b http://localhost:20003
    $ go run ./cmd/client/main.go

#### Option 2

    $ ulimit -Sn 65334 # must increase the NOFILE limit
    $ go run ./cmd/balancer/main.go -devservers 10 &
    $ go run ./cmd/client/main.go

### Monitoring

The balancer exposing prometheus metrics on `http://localhost:10000/metrics`.

    # HELP balancer_requests_seconds An hitogram observing time to server requests by backends
    # TYPE balancer_requests_seconds histogram

    # HELP balancer_retries_count A Counter to count the number of retries to each backend
    # TYPE balancer_retries_count counter

#### Examples of prometheus queries

Latency:

    sum(rate(balancer_requests_seconds_bucket{le="0.25"}[5m])) by (backend)
    /
    sum(rate(balancer_requests_seconds_count[5m])) by (backend)

Retries Rate:

    sum(rate(balancer_retries_count[5m]) by (backend)
