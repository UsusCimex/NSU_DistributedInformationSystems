package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"manager/models"
	"manager/store"
	"net/http"

	"github.com/google/uuid"
)

const workerURL = "http://worker:8080/internal/api/worker/hash/crack/task"

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
		partCount := request.MaxLength * 2
		log.Printf("Starting new task distribution. Hash: %s, Parts: %d", request.Hash, partCount)
		store.GlobalTaskStorage.SetPartCount(request.Hash, partCount)

		for i := 1; i <= partCount; i++ {
			task := models.CrackTaskRequest{
				Hash:       request.Hash,
				MaxLength:  request.MaxLength,
				PartNumber: i,
				PartCount:  partCount,
			}
			log.Printf("Sending part %d/%d for hash %s to worker", i, partCount, request.Hash)
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
	jsonData, err := json.Marshal(task)
	if err != nil {
		log.Printf("Failed to marshal task for hash %s: %v", task.Hash, err)
		return
	}

	log.Printf("Sending task to worker. Hash: %s, Part: %d/%d",
		task.Hash, task.PartNumber, task.PartCount)

	resp, err := http.Post(workerURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to send task to worker. Hash: %s, Error: %v", task.Hash, err)
		store.GlobalTaskStorage.UpdateStatus(task.Hash, "ERROR", []string{"Failed to send to worker"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Worker returned error status %d for hash %s", resp.StatusCode, task.Hash)
		store.GlobalTaskStorage.UpdateStatus(task.Hash, "ERROR", []string{"Worker returned error"})
	}
}
