package pool

import "worker/config"

type WorkerPool struct {
	pool chan struct{}
}

func New(cfg *config.Config) *WorkerPool {
	return &WorkerPool{
		pool: make(chan struct{}, cfg.MaxWorkers),
	}
}

func (wp *WorkerPool) Acquire() bool {
	select {
	case wp.pool <- struct{}{}:
		return true
	default:
		return false
	}
}

func (wp *WorkerPool) Release() {
	<-wp.pool
}
