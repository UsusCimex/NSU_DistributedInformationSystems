package handlers

import (
	"common/utils"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"worker/cracker"
	"worker/models"
	"worker/pool"
)

const (
	managerURL = "http://manager:8080/internal/api/manager/hash/crack/result"
	maxRetries = 5
	retryDelay = time.Second
)

var md5Cracker = cracker.NewMD5Cracker()

func CreateCrackTaskHandler(workerPool *pool.WorkerPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			log.Printf("Invalid method %s for crack task", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var task models.CrackTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
			log.Printf("Failed to decode task request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if !workerPool.Acquire() {
			log.Printf("No available workers for hash %s (part %d/%d)",
				task.Hash, task.PartNumber, task.PartCount)
			http.Error(w, "Server is busy", http.StatusServiceUnavailable)
			return
		}

		go func() {
			defer workerPool.Release()

			log.Printf("Starting crack attempt for hash %s (part %d/%d)",
				task.Hash, task.PartNumber, task.PartCount)

			result, err := md5Cracker.Crack(task)
			if err != nil {
				log.Printf("Crack failed for hash %s (part %d/%d): %v",
					task.Hash, task.PartNumber, task.PartCount, err)
				sendResult(models.CrackTaskResult{
					Hash:       task.Hash,
					Result:     "",
					PartNumber: task.PartNumber,
				}, r.Host)
				return
			}

			log.Printf("Found result for hash %s: %s (part %d/%d)",
				task.Hash, result, task.PartNumber, task.PartCount)

			sendResult(models.CrackTaskResult{
				Hash:       task.Hash,
				Result:     result,
				PartNumber: task.PartNumber,
			}, r.Host)
		}()

		w.WriteHeader(http.StatusOK)
	}
}

func sendResult(result models.CrackTaskResult, workerURL string) {
	req := utils.SendRequest{
		URL:     managerURL,
		Payload: result,
	}

	cfg := utils.SendConfig{
		MaxRetries:    maxRetries,
		Delay:         retryDelay,
		SuccessStatus: http.StatusOK,
	}

	if err := utils.RetryingSend(req, cfg); err != nil {
		log.Printf("Failed to send result for hash %s: %v", result.Hash, err)
	}
}
