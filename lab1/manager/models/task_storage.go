package models

import (
	"log"
	"sync"
)

type HashCrackStatus struct {
	Status string `json:"status"`
	Result string `json:"result"`
}

type TaskStorage struct {
	requestToHash map[string]string         // requestId -> hash
	hashToStatus  map[string]StatusResponse // hash -> status
	partResults   map[string]map[int]string // hash -> (partNumber -> result)
	partCounts    map[string]int            // hash -> expected number of parts
	mu            sync.RWMutex
}

func NewTaskStorage() *TaskStorage {
	return &TaskStorage{
		requestToHash: make(map[string]string),
		hashToStatus:  make(map[string]StatusResponse),
		partResults:   make(map[string]map[int]string),
		partCounts:    make(map[string]int),
	}
}

// Возвращает true, если хэш добавлен впервые и его надо крякнуть
func (ts *TaskStorage) AddTask(requestId string, hash string) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	log.Printf("[TaskStorage] Adding new task. RequestID: %s, Hash: %s", requestId, hash)

	if status, exists := ts.hashToStatus[hash]; exists {
		ts.requestToHash[requestId] = hash
		if status.Status == "DONE" {
			log.Printf("[TaskStorage] Hash %s already cracked, reusing result", hash)
			return false
		}
		if status.Status == "IN_PROGRESS" {
			log.Printf("[TaskStorage] Hash %s is already being processed, adding request to queue", hash)
			return false
		}
	}

	ts.requestToHash[requestId] = hash
	ts.hashToStatus[hash] = StatusResponse{
		Status: "IN_PROGRESS",
	}
	log.Printf("[TaskStorage] Started new task for hash %s", hash)
	return true
}

func (ts *TaskStorage) GetStatus(requestId string) (StatusResponse, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	hash, exists := ts.requestToHash[requestId]
	if !exists {
		log.Printf("[TaskStorage] Request ID %s not found", requestId)
		return StatusResponse{}, false
	}

	status, exists := ts.hashToStatus[hash]
	log.Printf("[TaskStorage] Status for request %s (hash %s): %s %v", requestId, hash, status.Status, status.Data)
	return status, exists
}

func (ts *TaskStorage) UpdateStatus(hash string, status string, result []string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if currentStatus, exists := ts.hashToStatus[hash]; exists {
		currentStatus.Status = status
		currentStatus.Data = result
		ts.hashToStatus[hash] = currentStatus
	}
}

func (ts *TaskStorage) SetPartCount(hash string, count int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	log.Printf("[TaskStorage] Setting part count for hash %s: %d parts", hash, count)
	if _, exists := ts.partCounts[hash]; !exists {
		ts.partCounts[hash] = count
		ts.partResults[hash] = make(map[int]string)
	}
}

func (ts *TaskStorage) AddPartResult(hash string, partNumber int, result string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	log.Printf("[TaskStorage] Adding result for hash %s part %d: %s",
		hash, partNumber, result)

	if _, exists := ts.partResults[hash]; !exists {
		ts.partResults[hash] = make(map[int]string)
	}
	ts.partResults[hash][partNumber] = result

	var successfulResults []string
	for _, res := range ts.partResults[hash] {
		if res != "" {
			successfulResults = append(successfulResults, res)
		}
	}

	if len(successfulResults) > 0 {
		log.Printf("[TaskStorage] Hash %s cracked successfully. Found matches: %v",
			hash, successfulResults)
		ts.hashToStatus[hash] = StatusResponse{
			Status: "DONE",
			Data:   successfulResults,
		}
		return
	}

	log.Printf("[TaskStorage] Received %d/%d parts for hash %s",
		len(ts.partResults[hash]), ts.partCounts[hash], hash)

	// Проверяем завершение только если все части обработаны и нет успешных результатов
	if len(ts.partResults[hash]) == ts.partCounts[hash] && len(successfulResults) == 0 {
		log.Printf("[TaskStorage] Failed to crack hash %s", hash)
		ts.hashToStatus[hash] = StatusResponse{
			Status: "FAIL",
			Data:   []string{},
		}
	}
}
