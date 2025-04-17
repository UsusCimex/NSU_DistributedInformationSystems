package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"common/constants"
	"common/logger"
	"common/models"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// CrackRequest представляет ожидаемое JSON-тело для запроса "crack".
type CrackRequest struct {
	Hash      string `json:"hash"`
	MaxLength int    `json:"maxLength"`
}

// CrackResponse представляет JSON-ответ на успешный запрос crack (возвращает ID для отслеживания запроса).
type CrackResponse struct {
	RequestId string `json:"requestId"`
}

// StatusResponse представляет JSON-ответ на запрос о статусе.
type StatusResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

// RegisterHandlers устанавливает HTTP обработчики для API взлома хешей.
func RegisterHandlers(mux *http.ServeMux, coll *mongo.Collection) {
	mux.HandleFunc("/api/hash/crack", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleCrack(w, r, coll)
	})
	mux.HandleFunc("/api/hash/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleStatus(w, r, coll)
	})
}

// handleCrack обрабатывает запрос на взлом заданного хеша.
func handleCrack(w http.ResponseWriter, r *http.Request, coll *mongo.Collection) {
	var req CrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log("API", "Ошибка декодирования запроса: "+err.Error())
		http.Error(w, "Bad request: invalid JSON", http.StatusBadRequest)
		return
	}
	if req.MaxLength < constants.MinMaxLength || req.MaxLength > constants.MaxMaxLength {
		http.Error(w, fmt.Sprintf("MaxLength must be between %d and %d", constants.MinMaxLength, constants.MaxMaxLength), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.ContextTimeout)
	defer cancel()

	// Если задача для этого хеша уже существует, возвращаем её requestId
	var existing models.HashTask
	err := coll.FindOne(ctx, bson.M{"hash": req.Hash}).Decode(&existing)
	if err == nil {
		json.NewEncoder(w).Encode(CrackResponse{RequestId: existing.RequestId})
		return
	}

	requestId := uuid.New().String()
	now := time.Now()

	totalCandidates := math.Pow(float64(constants.AlphabetSize), float64(req.MaxLength))
	numSubTasks := int(math.Ceil(totalCandidates / constants.MaxCandidatesPerSubTask))
	if numSubTasks < 1 {
		numSubTasks = 1
	}

	subTasks := make([]models.SubTask, numSubTasks)
	for i := 0; i < numSubTasks; i++ {
		subTasks[i] = models.SubTask{
			Hash:          req.Hash,
			SubTaskNumber: i + 1,
			Status:        "RECEIVED",
			CreatedAt:     now,
			UpdatedAt:     now,
		}
	}

	taskDoc := models.HashTask{
		RequestId:          requestId,
		Hash:               req.Hash,
		MaxLength:          req.MaxLength,
		Status:             "IN_PROGRESS",
		SubTaskCount:       numSubTasks,
		CompletedTaskCount: 0,
		Result:             "",
		SubTasks:           subTasks,
		CreatedAt:          now,
	}

	_, err = coll.InsertOne(ctx, taskDoc)
	if err != nil {
		logger.Log("API", fmt.Sprintf("Ошибка вставки задачи для хэша %s: %v", req.Hash, err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	logger.LogHash("API", req.Hash, fmt.Sprintf("Новая задача создана (RequestId=%s, подзадач: %d)", requestId, numSubTasks))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CrackResponse{RequestId: requestId})
}

// handleStatus возвращает статус задачи взлома по её requestId.
func handleStatus(w http.ResponseWriter, r *http.Request, coll *mongo.Collection) {
	requestId := r.URL.Query().Get("requestId")
	if requestId == "" {
		http.Error(w, "requestId parameter is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.ContextTimeout)
	defer cancel()

	var task models.HashTask
	err := coll.FindOne(ctx, bson.M{"requestId": requestId}).Decode(&task)
	if err != nil {
		logger.Log("API", "Задача с requestId "+requestId+" не найдена")
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	var resp StatusResponse
	switch task.Status {
	case "IN_PROGRESS":
		percent := float64(task.CompletedTaskCount) / float64(task.SubTaskCount) * 100.0
		resp = StatusResponse{Status: "IN_PROGRESS", Data: percent}
	case "DONE":
		resp = StatusResponse{Status: "DONE", Data: task.Result}
	case "FAIL":
		resp = StatusResponse{Status: "FAIL", Data: "Хэш не был расшифрован"}
	default:
		resp = StatusResponse{Status: "IN_PROGRESS", Data: 0.0}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// StartHTTPServer инициализирует и запускает HTTP-сервер для обработки API-запросов.
func StartHTTPServer(coll *mongo.Collection) {
	mux := http.NewServeMux()
	RegisterHandlers(mux, coll)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.Log("HTTP", "HTTP-сервер запущен на порту "+port)
	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		logger.Log("HTTP", "Ошибка работы HTTP-сервера: "+err.Error())
		panic(err)
	}
}
