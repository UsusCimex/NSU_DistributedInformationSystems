package handlers

import (
	"encoding/json"
	"manager/dispatcher"
	"manager/models"
	"manager/store"
	"net/http"
)

func ResultHandler(dispatcher *dispatcher.TaskDispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var result models.CrackTaskResult
		if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if workerURL := dispatcher.GetWorkerByPart(result.Hash, result.PartNumber); workerURL != "" {
			dispatcher.DecrementWorkerTasks(workerURL)
		}

		store.GlobalTaskStorage.AddPartResult(result.Hash, result.PartNumber, result.Result)
		w.WriteHeader(http.StatusOK)
	}
}
