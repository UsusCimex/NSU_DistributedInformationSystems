package handlers

import (
	"encoding/json"
	"manager/store"
	"net/http"
)

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestId := r.URL.Query().Get("requestId")
	if requestId == "" {
		http.Error(w, "Missing requestId parameter", http.StatusBadRequest)
		return
	}

	status, exists := store.GlobalTaskStorage.GetStatus(requestId)
	if !exists {
		http.Error(w, "Request not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
