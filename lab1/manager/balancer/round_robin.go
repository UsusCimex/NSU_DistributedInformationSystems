package balancer

import (
	"os"
	"strings"
	"sync"
)

type RoundRobin struct {
	workers []string
	current int
	mu      sync.Mutex
}

var LoadBalancer *RoundRobin

func Init() {
	workerURLs := os.Getenv("WORKER_URLS")
	if workerURLs == "" {
		workerURLs = "http://worker:8080"
	}

	workers := strings.Split(workerURLs, ",")
	LoadBalancer = &RoundRobin{
		workers: workers,
		current: 0,
	}
}

func (rb *RoundRobin) GetNextWorker() string {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	worker := rb.workers[rb.current]
	rb.current = (rb.current + 1) % len(rb.workers)
	return worker
}
