package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	MaxWorkers int
	WorkerURL  string
	ManagerURL string
	Port       string
}

const (
	defaultMaxWorkers = 10
	defaultPort       = "8080"
)

func Load() *Config {
	maxWorkers := defaultMaxWorkers
	if val := os.Getenv("MAX_WORKERS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxWorkers = parsed
		} else {
			log.Printf("Warning: Invalid MAX_WORKERS value: %s, using default: %d", val, defaultMaxWorkers)
		}
	} else {
		log.Printf("Info: MAX_WORKERS not set, using default: %d", defaultMaxWorkers)
	}

	workerURL := os.Getenv("WORKER_URL")
	managerURL := os.Getenv("MANAGER_URL")
	if managerURL == "" {
		managerURL = "http://manager:8080"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	return &Config{
		MaxWorkers: maxWorkers,
		WorkerURL:  workerURL,
		ManagerURL: managerURL,
		Port:       port,
	}
}
