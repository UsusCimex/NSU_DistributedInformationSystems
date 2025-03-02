package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"worker/cracker"
	"worker/models"
)

const managerURL = "http://manager:8080/internal/api/manager/hash/crack/request"

var md5Cracker = cracker.NewMD5Cracker()

func CrackTaskHandler(w http.ResponseWriter, r *http.Request) {
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

	log.Printf("Received crack task. Hash: %s, Part: %d/%d",
		task.Hash, task.PartNumber, task.PartCount)

	go func() {
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
}

func sendResult(result models.CrackTaskResult) {
	jsonData, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to marshal result for hash %s: %v", result.Hash, err)
		return
	}

	log.Printf("Sending result for hash %s back to manager", result.Hash)

	resp, err := http.Post(managerURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to send result to manager for hash %s: %v", result.Hash, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Manager returned error status %d for hash %s", resp.StatusCode, result.Hash)
	}
}
