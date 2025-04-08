package main

import "time"

type Task struct {
	RequestId string    `bson:"requestId"`
	Hash      string    `bson:"hash"`
	CreatedAt time.Time `bson:"createdAt"`
}

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

type SubTask struct {
	ID            string    `bson:"id"`
	Hash          string    `bson:"hash"`
	Status        string    `bson:"status"` // RECEIVED, PUBLISHED, WORKING, COMPLETE
	WorkerId      string    `bson:"workerId,omitempty"`
	Heartbeat     time.Time `bson:"heartbeat,omitempty"`
	CreatedAt     time.Time `bson:"createdAt"`
	UpdatedAt     time.Time `bson:"updatedAt"`
	SubTaskNumber int       `bson:"subTaskNumber"`
}
