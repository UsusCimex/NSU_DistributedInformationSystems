package dispatcher

import (
	"fmt"
	"log"
	"manager/models"
	"manager/queue"
	"net/http"
	"sync"
	"time"

	"common/utils"
	"manager/balancer"
)

type TaskDispatcher struct {
	taskQueue    *queue.TaskQueue
	partToWorker map[string]map[int]string // hash -> partNumber -> workerURL
	mu           sync.RWMutex
}

func NewTaskDispatcher(taskQueue *queue.TaskQueue) *TaskDispatcher {
	return &TaskDispatcher{
		taskQueue:    taskQueue,
		partToWorker: make(map[string]map[int]string),
	}
}

func (d *TaskDispatcher) Start() {
	go d.dispatchTasks()
}

func (d *TaskDispatcher) dispatchTasks() {
	for {
		task := d.taskQueue.Pop()
		worker := balancer.LoadBalancer.GetNextWorker()

		if worker != nil {
			log.Printf("Dispatching task for hash %s (part %d/%d) to worker %s",
				task.Hash, task.PartNumber, task.PartCount, worker.URL)
			go d.sendTaskToWorker(worker.URL, *task)
		} else {
			// На всякий случай, хотя такого не должно происходить
			log.Printf("No worker available, returning task to queue")
			d.taskQueue.Push(*task)
		}
	}
}

func (d *TaskDispatcher) sendTaskToWorker(workerURL string, task models.CrackTaskRequest) {
	d.mu.Lock()
	if _, exists := d.partToWorker[task.Hash]; !exists {
		d.partToWorker[task.Hash] = make(map[int]string)
	}
	d.partToWorker[task.Hash][task.PartNumber] = workerURL
	d.mu.Unlock()

	req := utils.SendRequest{
		URL:     fmt.Sprintf("%s/internal/api/worker/hash/crack/task", workerURL),
		Payload: task,
	}

	cfg := utils.SendConfig{
		MaxRetries:      3,
		Delay:           time.Second * 2,
		SuccessStatus:   http.StatusOK,
		RetryOnStatuses: []int{http.StatusServiceUnavailable},
	}

	if err := utils.RetryingSend(req, cfg); err != nil {
		log.Printf("Failed to send task to worker %s: %v", workerURL, err)
		balancer.LoadBalancer.TaskCompleted(workerURL) // Освобождаем слот в случае ошибки
		d.taskQueue.Push(task)                         // Возвращаем задачу в очередь
	}
}

func (d *TaskDispatcher) DecrementWorkerTasks(workerURL string) {
	balancer.LoadBalancer.TaskCompleted(workerURL)
}

func (d *TaskDispatcher) GetWorkerByPart(hash string, partNumber int) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if parts, exists := d.partToWorker[hash]; exists {
		return parts[partNumber]
	}
	return ""
}
