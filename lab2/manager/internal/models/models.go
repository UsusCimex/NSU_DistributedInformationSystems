package models

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

// SubTask описывает подзадачу; идентифицируется по hash и subTaskNumber.
type SubTask struct {
	Hash          string    `bson:"hash"`
	Status        string    `bson:"status"` // RECEIVED, PUBLISHED, WORKING, COMPLETE
	WorkerId      string    `bson:"workerId,omitempty"`
	Heartbeat     time.Time `bson:"heartbeat,omitempty"`
	CreatedAt     time.Time `bson:"createdAt"`
	UpdatedAt     time.Time `bson:"updatedAt"`
	SubTaskNumber int       `bson:"subTaskNumber"`
}

// TaskMessage описывает сообщение для публикации задачи в RabbitMQ.
type TaskMessage struct {
	Hash          string `json:"hash"`
	MaxLength     int    `json:"maxLength"`
	SubTaskNumber int    `json:"subTaskNumber"`
	SubTaskCount  int    `json:"subTaskCount"`
}

// ResultMessage описывает сообщение результата, полученное от Worker-а.
type ResultMessage struct {
	Hash          string `json:"hash"`
	SubTaskNumber int    `json:"subTaskNumber"`
	WorkerId      string `json:"workerId"`
	Result        string `json:"result"`
}
