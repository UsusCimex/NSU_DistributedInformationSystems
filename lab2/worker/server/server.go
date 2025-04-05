package server

import (
	"log"
	"net/http"
	"worker/config"
	"worker/handlers"
	"worker/pool"
)

func Start(cfg *config.Config, workerPool *pool.WorkerPool) {
	http.HandleFunc("/internal/api/worker/hash/crack/task", handlers.CreateCrackTaskHandler(workerPool))

	log.Printf("Starting worker server on port %s with %d max workers", cfg.Port, cfg.MaxWorkers)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatal(err)
	}
}
