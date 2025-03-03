package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"manager/balancer"
	"manager/models"
	"manager/store"
	"net/http"
	"time"

	"common/utils"

	"github.com/google/uuid"
)

const (
	partCoefficient = 50
	maxRetries      = 1_000_000
	initialDelay    = 1 * time.Second
	maxDelay        = 2 * time.Minute
)

func CrackHashHandler(w http.ResponseWriter, r *http.Request) {
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
	log.Printf("Received new crack request. ID: %s, Hash: %s, MaxLength: %d",
		requestId, request.Hash, request.MaxLength)

	needWorker := store.GlobalTaskStorage.AddTask(requestId, request.Hash)
	if needWorker {
		partCount := request.MaxLength * partCoefficient
		log.Printf("Starting new task distribution. Hash: %s, Parts: %d", request.Hash, partCount)
		store.GlobalTaskStorage.SetPartCount(request.Hash, partCount)

		for i := 1; i <= partCount; i++ {
			task := models.CrackTaskRequest{
				Hash:       request.Hash,
				MaxLength:  request.MaxLength,
				PartNumber: i,
				PartCount:  partCount,
			}
			go sendToWorker(task)
		}
	} else {
		log.Printf("Task for hash %s already exists, returning existing request ID", request.Hash)
	}

	response := models.HashCrackResponse{
		RequestId: requestId,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func sendToWorker(task models.CrackTaskRequest) {
	req := utils.SendRequest{
		URL: fmt.Sprintf("%s/internal/api/worker/hash/crack/task",
			balancer.LoadBalancer.GetNextWorker()),
		Payload: task,
	}

	cfg := utils.SendConfig{
		MaxRetries:      maxRetries,
		InitialDelay:    initialDelay,
		MaxDelay:        maxDelay,
		SuccessStatus:   http.StatusOK,
		RetryOnStatuses: []int{http.StatusServiceUnavailable},
	}

	if err := utils.RetryingSend(req, cfg); err != nil {
		log.Printf("Failed to send task for hash %s, part %d: %v",
			task.Hash, task.PartNumber, err)
		store.GlobalTaskStorage.UpdateStatus(task.Hash, "ERROR",
			[]string{fmt.Sprintf("Failed to send part %d: %v",
				task.PartNumber, err)})
	}
}
