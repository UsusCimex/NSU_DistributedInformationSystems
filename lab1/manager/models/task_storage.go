package models

import (
	"fmt"
	"log"
	"sync"
)

type HashCrackStatus struct {
	Status string `json:"status"`
	Result string `json:"result"`
}

type TaskStorage struct {
	requestToHash map[string]string         // requestId -> hash
	hashToStatus  map[string]StatusResponse // hash -> task status
	partResults   map[string]map[int]string // hash -> (part number -> result)
	partCounts    map[string]int            // hash -> expected parts count
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

// AddTask returns true if hash is added for the first time and needs to be cracked
func (ts *TaskStorage) AddTask(requestId string, hash string) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	log.Printf("[TaskStorage] Adding new task. RequestID: %s, Hash: %s", requestId, hash)
	ts.requestToHash[requestId] = hash

	if status, exists := ts.hashToStatus[hash]; exists {
		if status.Status == "DONE" {
			log.Printf("[TaskStorage] Hash %s already processed, returning result", hash)
			return false
		}
		if status.Status == "IN_PROGRESS" {
			log.Printf("[TaskStorage] Hash %s already in progress", hash)
			return false
		}
	}

	ts.hashToStatus[hash] = StatusResponse{
		Status: "IN_PROGRESS",
		Data:   []string{"0%"},
	}
	log.Printf("[TaskStorage] Started processing for hash %s", hash)
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

	if _, exists := ts.partCounts[hash]; !exists {
		ts.partCounts[hash] = count
		ts.partResults[hash] = make(map[int]string)
	}
}

func (ts *TaskStorage) AddPartResult(hash string, partNumber int, result string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if _, exists := ts.partResults[hash]; !exists {
		ts.partResults[hash] = make(map[int]string)
	}
	ts.partResults[hash][partNumber] = result

	progress := float64(len(ts.partResults[hash])) / float64(ts.partCounts[hash]) * 100

	if result != "" {
		var successfulResults []string
		for _, res := range ts.partResults[hash] {
			if res != "" {
				successfulResults = append(successfulResults, res)
			}
		}
		if len(successfulResults) > 0 {
			ts.hashToStatus[hash] = StatusResponse{
				Status: "DONE",
				Data:   successfulResults,
			}
			return
		}
	}

	if ts.hashToStatus[hash].Status != "DONE" {
		ts.hashToStatus[hash] = StatusResponse{
			Status: "IN_PROGRESS",
			Data:   []string{fmt.Sprintf("%.1f%%", progress)},
		}
	}

	if len(ts.partResults[hash]) == ts.partCounts[hash] && ts.hashToStatus[hash].Status != "DONE" {
		ts.hashToStatus[hash] = StatusResponse{
			Status: "FAIL",
			Data:   []string{},
		}
	}
}
