package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	MaxWorkers int
}

const defaultMaxWorkers = 10

func Load() Config {
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

	return Config{
		MaxWorkers: maxWorkers,
	}
}
