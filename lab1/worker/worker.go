package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
	"worker/config"
	"worker/handlers"
)

func registerWithManager() error {
	cfg := config.Load()
	registration := struct {
		WorkerURL  string `json:"workerUrl"`
		MaxWorkers int    `json:"maxWorkers"`
	}{
		WorkerURL:  os.Getenv("WORKER_URL"),
		MaxWorkers: cfg.MaxWorkers,
	}

	body, err := json.Marshal(registration)
	if err != nil {
		return err
	}

	managerURL := os.Getenv("MANAGER_URL")
	if managerURL == "" {
		managerURL = "http://manager:8080"
	}

	for {
		resp, err := http.Post(managerURL+"/internal/api/worker/register",
			"application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Printf("Failed to register with manager: %v, retrying in 5 seconds...", err)
			time.Sleep(5 * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			log.Printf("Registration failed with status %d, retrying in 5 seconds...", resp.StatusCode)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Printf("Successfully registered with manager at %s", managerURL)
		return nil
	}
}

func main() {
	cfg := config.Load()

	if err := registerWithManager(); err != nil {
		log.Fatal("Failed to register with manager:", err)
	}

	workerPool := make(chan struct{}, cfg.MaxWorkers)
	http.HandleFunc("/internal/api/worker/hash/crack/task", handlers.CreateCrackTaskHandler(workerPool))

	log.Printf("Starting worker server on :8080 with %d max concurrent workers", cfg.MaxWorkers)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
