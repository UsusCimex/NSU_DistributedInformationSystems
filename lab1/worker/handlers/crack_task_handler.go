package handlers

import (
	"common/utils"
	"encoding/json"
	"log"
	"net/http"
	"time"
	"worker/cracker"
	"worker/models"
)

const (
	managerURL = "http://manager:8080/internal/api/manager/hash/crack/request"

	maxRetries   = 10
	initialDelay = 1 * time.Second
	maxDelay     = 5 * time.Minute
)

var md5Cracker = cracker.NewMD5Cracker()

func CreateCrackTaskHandler(workerPool chan struct{}) http.HandlerFunc {
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

		select {
		case workerPool <- struct{}{}: // Пытаемся получить слот в пуле
			log.Printf("Acquired worker slot. Processing task for hash %s (part %d/%d)",
				task.Hash, task.PartNumber, task.PartCount)

			go func() {
				defer func() { <-workerPool }() // Освобождаем слот после завершения

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
					})
					return
				}

				log.Printf("Found result for hash %s: %s (part %d/%d)",
					task.Hash, result, task.PartNumber, task.PartCount)

				sendResult(models.CrackTaskResult{
					Hash:       task.Hash,
					Result:     result,
					PartNumber: task.PartNumber,
				})
			}()

			w.WriteHeader(http.StatusOK)

		default: // Если нет свободных слотов
			log.Printf("No available workers for hash %s (part %d/%d)",
				task.Hash, task.PartNumber, task.PartCount)
			http.Error(w, "Server is busy", http.StatusServiceUnavailable)
		}
	}
}

func sendResult(result models.CrackTaskResult) {
	req := utils.SendRequest{
		URL:     managerURL,
		Payload: result,
	}

	cfg := utils.SendConfig{
		MaxRetries:    maxRetries,
		InitialDelay:  initialDelay,
		MaxDelay:      maxDelay,
		SuccessStatus: http.StatusOK,
	}

	if err := utils.RetryingSend(req, cfg); err != nil {
		log.Printf("Failed to send result for hash %s: %v", result.Hash, err)
	}
}
