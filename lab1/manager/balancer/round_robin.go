package balancer

import (
	"log"
	"sync"
)

type WorkerInfo struct {
	URL         string
	MaxWorkers  int
	ActiveTasks int
}

type RoundRobin struct {
	workers     []*WorkerInfo
	mu          sync.Mutex
	workerReady chan struct{} // Канал для сигнализации о доступных слотах
}

var LoadBalancer *RoundRobin

func Init() {
	LoadBalancer = &RoundRobin{
		workers:     make([]*WorkerInfo, 0),
		workerReady: make(chan struct{}, 100),
	}
}

func (rb *RoundRobin) RegisterWorker(url string, maxWorkers int) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	log.Printf("Registering new worker: %s with max tasks: %d", url, maxWorkers)
	rb.workers = append(rb.workers, &WorkerInfo{
		URL:         url,
		MaxWorkers:  maxWorkers,
		ActiveTasks: 0,
	})
	log.Printf("Total registered workers: %d", len(rb.workers))

	// Изначально все слоты считаются свободными
	for i := 0; i < maxWorkers; i++ {
		rb.workerReady <- struct{}{}
	}
}

func (rb *RoundRobin) GetNextWorker() *WorkerInfo {
	<-rb.workerReady // Ждём появления свободного слота

	rb.mu.Lock()
	defer rb.mu.Unlock()

	var selectedWorker *WorkerInfo
	for _, worker := range rb.workers {
		if worker.ActiveTasks < worker.MaxWorkers &&
			(selectedWorker == nil || worker.ActiveTasks < selectedWorker.ActiveTasks) {
			selectedWorker = worker
		}
	}

	if selectedWorker != nil {
		selectedWorker.ActiveTasks++
	}
	return selectedWorker
}

func (rb *RoundRobin) TaskCompleted(workerURL string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for _, worker := range rb.workers {
		if worker.URL == workerURL {
			if worker.ActiveTasks > 0 {
				worker.ActiveTasks--
				rb.workerReady <- struct{}{} // Сигнал о освобождении слота
			}
			break
		}
	}
}
