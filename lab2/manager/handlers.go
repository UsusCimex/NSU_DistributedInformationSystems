package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

// CrackRequest для POST /api/hash/crack.
type CrackRequest struct {
	Hash      string `json:"hash"`
	MaxLength int    `json:"maxLength"`
}

// CrackResponse для POST /api/hash/crack.
type CrackResponse struct {
	RequestId string `json:"requestId"`
}

// StatusResponse для GET /api/hash/status.
type StatusResponse struct {
	Status string      `json:"status"` // DONE, IN_PROGRESS, FAIL
	Data   interface{} `json:"data"`   // либо найденная строка, либо процент выполнения
}

// handleCrack создаёт новую задачу для взлома хэша.
func handleCrack(w http.ResponseWriter, r *http.Request) {
	var req CrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Ошибка декодирования запроса: %v", err)
		http.Error(w, "Неверный запрос", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var existingTask HashTask
	err := hashTaskCollection.FindOne(ctx, bson.M{"hash": req.Hash}).Decode(&existingTask)
	if err == nil {
		resp := CrackResponse{RequestId: existingTask.RequestId}
		json.NewEncoder(w).Encode(resp)
		return
	}

	requestId := uuid.New().String()
	numSubTasks := 100

	var subTasks []SubTask
	now := time.Now()
	for i := 0; i < numSubTasks; i++ {
		subTask := SubTask{
			Hash:          req.Hash,
			Status:        "RECEIVED",
			CreatedAt:     now,
			UpdatedAt:     now,
			SubTaskNumber: i + 1,
		}
		subTasks = append(subTasks, subTask)
	}

	hashTask := HashTask{
		RequestId:          requestId,
		Hash:               req.Hash,
		MaxLength:          req.MaxLength,
		Status:             "IN_PROGRESS",
		SubTaskCount:       numSubTasks,
		CompletedTaskCount: 0,
		SubTasks:           subTasks,
		CreatedAt:          now,
	}

	_, err = hashTaskCollection.InsertOne(ctx, hashTask)
	if err != nil {
		log.Printf("Ошибка при сохранении задачи: %v", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	log.Printf("Создана новая задача: %s", requestId)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CrackResponse{RequestId: requestId})
}

// handleStatus возвращает статус задачи по requestId.
func handleStatus(w http.ResponseWriter, r *http.Request) {
	requestId := r.URL.Query().Get("requestId")
	if requestId == "" {
		http.Error(w, "requestId обязателен", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var task HashTask
	err := hashTaskCollection.FindOne(ctx, bson.M{"requestId": requestId}).Decode(&task)
	if err != nil {
		http.Error(w, "Задача не найдена", http.StatusNotFound)
		return
	}

	var resp StatusResponse
	switch task.Status {
	case "IN_PROGRESS":
		percent := float64(task.CompletedTaskCount) / float64(task.SubTaskCount) * 100
		resp = StatusResponse{Status: "IN_PROGRESS", Data: percent}
	case "DONE":
		resp = StatusResponse{Status: "DONE", Data: task.Result}
	case "FAIL":
		resp = StatusResponse{Status: "ERROR", Data: "Задача завершилась с ошибкой"}
	default:
		resp = StatusResponse{Status: "IN_PROGRESS", Data: 0}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
