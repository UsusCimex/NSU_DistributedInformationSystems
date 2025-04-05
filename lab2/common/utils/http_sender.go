package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type SendConfig struct {
	MaxRetries      int
	Delay           time.Duration
	SuccessStatus   int
	RetryOnStatuses []int
}

type SendRequest struct {
	URL     string
	Payload interface{}
}

// TrySend пытается отправить запрос один раз
func TrySend(req SendRequest) error {
	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := http.Post(req.URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

func RetryingSend(req SendRequest, cfg SendConfig) error {
	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Retrying request to %s. Attempt %d/%d",
				req.URL, attempt+1, cfg.MaxRetries)
			time.Sleep(cfg.Delay)
		}

		resp, err := http.Post(req.URL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("network error: %w", err)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode == cfg.SuccessStatus {
			return nil
		}

		shouldRetry := false
		for _, status := range cfg.RetryOnStatuses {
			if resp.StatusCode == status {
				shouldRetry = true
				break
			}
		}

		if shouldRetry {
			lastErr = fmt.Errorf("server returned retriable status %d", resp.StatusCode)
			continue
		}

		return fmt.Errorf("server returned non-retriable status %d", resp.StatusCode)
	}

	return fmt.Errorf("failed after %d attempts, last error: %v", cfg.MaxRetries, lastErr)
}
