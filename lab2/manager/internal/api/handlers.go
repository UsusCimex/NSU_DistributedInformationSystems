package api

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"manager/internal/models"
)

// APIHandler инкапсулирует зависимости HTTP-обработчиков.
type APIHandler struct {
	coll *mongo.Collection
}

func NewAPIHandler(coll *mongo.Collection) *APIHandler {
	return &APIHandler{coll: coll}
}

// RegisterHandlers регистрирует обработчики API.
func RegisterHandlers(mux *http.ServeMux, coll *mongo.Collection) {
	h := NewAPIHandler(coll)
	mux.HandleFunc("/api/hash/crack", h.handleCrack)
	mux.HandleFunc("/api/hash/status", h.handleStatus)
}

func (h *APIHandler) handleCrack(w http.ResponseWriter, r *http.Request) {
	var req CrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Ошибка декодирования запроса: %v", err)
		http.Error(w, "Неверный запрос", http.StatusBadRequest)
		return
	}
	if (req.MaxLength < 1) || (req.MaxLength > 7) {
		http.Error(w, "Max length between 1 and 7", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existing models.HashTask
	if err := h.coll.FindOne(ctx, bson.M{"hash": req.Hash}).Decode(&existing); err == nil {
		json.NewEncoder(w).Encode(CrackResponse{RequestId: existing.RequestId})
		return
	}

	requestId := uuid.New().String()
	now := time.Now()

	const alphabetSize = 36 // a-z и 0-9
	totalCandidates := math.Pow(float64(alphabetSize), float64(req.MaxLength))
	const maxCandidatesPerSubTask = 25_000_000.0
	numSubTasks := int(math.Ceil(totalCandidates / maxCandidatesPerSubTask))
	if numSubTasks < 1 {
		numSubTasks = 1
	}

	subTasks := make([]models.SubTask, numSubTasks)
	for i := 0; i < numSubTasks; i++ {
		subTasks[i] = models.SubTask{
			Hash:          req.Hash,
			Status:        "RECEIVED",
			CreatedAt:     now,
			UpdatedAt:     now,
			SubTaskNumber: i + 1,
		}
	}

	hashTask := models.HashTask{
		RequestId:          requestId,
		Hash:               req.Hash,
		MaxLength:          req.MaxLength,
		Status:             "IN_PROGRESS",
		SubTaskCount:       numSubTasks,
		CompletedTaskCount: 0,
		SubTasks:           subTasks,
		CreatedAt:          now,
	}

	if _, err := h.coll.InsertOne(ctx, hashTask); err != nil {
		log.Printf("Ошибка сохранения задачи: %v", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	log.Printf("Создана новая задача: %s, число подзадач: %d", requestId, numSubTasks)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CrackResponse{RequestId: requestId})
}

func (h *APIHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	requestId := r.URL.Query().Get("requestId")
	if requestId == "" {
		http.Error(w, "requestId обязателен", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var task models.HashTask
	if err := h.coll.FindOne(ctx, bson.M{"requestId": requestId}).Decode(&task); err != nil {
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

// Определения типов запросов/ответов.
type CrackRequest struct {
	Hash      string `json:"hash"`
	MaxLength int    `json:"maxLength"`
}

type CrackResponse struct {
	RequestId string `json:"requestId"`
}

type StatusResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}
