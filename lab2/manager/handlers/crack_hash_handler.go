package handlers

import (
	"encoding/json"
	"log"
	"manager/models"
	"manager/queue"
	"manager/store"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	partCoefficient = 50
	maxRetries      = 1_000_000
	initialDelay    = 1 * time.Second
	maxDelay        = 2 * time.Minute
)

func CrackHashHandler(taskQueue *queue.TaskQueue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			log.Printf("Invalid method %s for crack hash request", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request models.HashCrackRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			log.Printf("Failed to decode request: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		requestId := uuid.New().String()
		needWorker := store.GlobalTaskStorage.AddTask(requestId, request.Hash)

		if needWorker {
			partCount := request.MaxLength * partCoefficient
			store.GlobalTaskStorage.SetPartCount(request.Hash, partCount)

			for i := 1; i <= partCount; i++ {
				task := models.CrackTaskRequest{
					Hash:       request.Hash,
					MaxLength:  request.MaxLength,
					PartNumber: i,
					PartCount:  partCount,
				}
				taskQueue.Push(task)
			}
		}

		response := models.HashCrackResponse{
			RequestId: requestId,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
