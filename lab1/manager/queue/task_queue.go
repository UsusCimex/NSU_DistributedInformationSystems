package queue

import (
	"manager/models"
	"sync"
)

type TaskQueue struct {
	tasks    []models.CrackTaskRequest
	mu       sync.Mutex
	notEmpty *sync.Cond
}

func NewTaskQueue() *TaskQueue {
	q := &TaskQueue{}
	q.notEmpty = sync.NewCond(&q.mu)
	return q
}

func (q *TaskQueue) Push(task models.CrackTaskRequest) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.tasks = append(q.tasks, task)
	q.notEmpty.Signal()
}

func (q *TaskQueue) Pop() *models.CrackTaskRequest {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.tasks) == 0 {
		q.notEmpty.Wait()
	}

	task := q.tasks[0]
	q.tasks = q.tasks[1:]
	return &task
}
