package handlers

import (
	"encoding/json"
	"manager/balancer"
	"net/http"
)

func WorkerRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var registration struct {
		WorkerURL  string `json:"workerUrl"`
		MaxWorkers int    `json:"maxWorkers"`
	}

	if err := json.NewDecoder(r.Body).Decode(&registration); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	balancer.LoadBalancer.RegisterWorker(registration.WorkerURL, registration.MaxWorkers)
	w.WriteHeader(http.StatusOK)
}
