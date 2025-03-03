package main

import (
	"log"
	"net/http"
	"worker/config"
	"worker/handlers"
)

func main() {
	cfg := config.Load()
	workerPool := make(chan struct{}, cfg.MaxWorkers)

	http.HandleFunc("/internal/api/worker/hash/crack/task", handlers.CreateCrackTaskHandler(workerPool))

	log.Printf("Starting worker server on :8080 with %d max concurrent workers", cfg.MaxWorkers)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
