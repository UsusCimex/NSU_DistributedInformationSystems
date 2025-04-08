package main

import "time"

// HashTask – основная задача для взлома хэша.
type HashTask struct {
	RequestId          string    `bson:"requestId"`
	Hash               string    `bson:"hash"`
	MaxLength          int       `bson:"maxLength"`
	Status             string    `bson:"status"` // IN_PROGRESS, DONE, FAIL
	SubTaskCount       int       `bson:"subTaskCount"`
	CompletedTaskCount int       `bson:"completedTaskCount"`
	Result             string    `bson:"result,omitempty"`
	SubTasks           []SubTask `bson:"subTasks"`
	CreatedAt          time.Time `bson:"createdAt"`
}

// SubTask – подзадача, определяется по hash и subTaskNumber.
type SubTask struct {
	Hash          string    `bson:"hash"`
	Status        string    `bson:"status"` // RECEIVED, PUBLISHED, WORKING, COMPLETE
	WorkerId      string    `bson:"workerId,omitempty"`
	Heartbeat     time.Time `bson:"heartbeat,omitempty"`
	CreatedAt     time.Time `bson:"createdAt"`
	UpdatedAt     time.Time `bson:"updatedAt"`
	SubTaskNumber int       `bson:"subTaskNumber"`
}
