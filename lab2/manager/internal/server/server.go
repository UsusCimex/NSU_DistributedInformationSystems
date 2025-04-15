package server

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

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
	if req.MaxLength < 1 || req.MaxLength > 7 {
		http.Error(w, "MaxLength must be between 1 and 7", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Если задача для этого хеша уже существует, возвращаем её requestId
	var existing models.HashTask
	err := coll.FindOne(ctx, bson.M{"hash": req.Hash}).Decode(&existing)
	if err == nil {
		json.NewEncoder(w).Encode(CrackResponse{RequestId: existing.RequestId})
		return
	}

	// Создаем новую задачу взлома
	requestId := uuid.New().String()
	now := time.Now()

	// Определяем количество подзадач на основе размера пространства поиска
	const alphabetSize = 36 // 26 букв + 10 цифр
	totalCandidates := math.Pow(float64(alphabetSize), float64(req.MaxLength))
	const maxCandidatesPerSubTask = 15_000_000.0
	numSubTasks := int(math.Ceil(totalCandidates / maxCandidatesPerSubTask))
	if numSubTasks < 1 {
		numSubTasks = 1
	}

	// Инициализируем массив подзадач
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

	// Создаем документ HashTask
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var task models.HashTask
	err := coll.FindOne(ctx, bson.M{"requestId": requestId}).Decode(&task)
	if err != nil {
		logger.Log("API", "Задача с requestId "+requestId+" не найдена")
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Формируем ответ на основе статуса задачи
	var resp StatusResponse
	switch task.Status {
	case "IN_PROGRESS":
		// Вычисляем процент выполнения
		percent := float64(task.CompletedTaskCount) / float64(task.SubTaskCount) * 100.0
		resp = StatusResponse{Status: "IN_PROGRESS", Data: percent}
	case "DONE":
		// Задача выполнена и результат найден
		resp = StatusResponse{Status: "DONE", Data: task.Result}
	case "FAIL":
		// Задача выполнена, но результат не найден
		resp = StatusResponse{Status: "FAIL", Data: "Хэш не был расшифрован"}
	default:
		// Неожиданный статус
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
