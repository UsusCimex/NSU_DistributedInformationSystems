package config

import (
	"os"
	"strconv"
)

type Config struct {
	MaxWorkers int
}

func Load() Config {
	maxWorkers := 10
	if val := os.Getenv("MAX_WORKERS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxWorkers = parsed
		}
	}

	return Config{
		MaxWorkers: maxWorkers,
	}
}
