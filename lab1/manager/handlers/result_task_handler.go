package handlers

import (
	"encoding/json"
	"log"
	"manager/models"
	"manager/store"
	"net/http"
)

func ResultTaskHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("[ResultHandler] Invalid method %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var result models.CrackTaskResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		log.Printf("[ResultHandler] Failed to decode result: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("[ResultHandler] Received result for hash %s part %d: %s",
		result.Hash, result.PartNumber, result.Result)

	store.GlobalTaskStorage.AddPartResult(result.Hash, result.PartNumber, result.Result)
	log.Printf("[ResultHandler] Successfully processed result for hash %s part %d",
		result.Hash, result.PartNumber)

	w.WriteHeader(http.StatusOK)
}
