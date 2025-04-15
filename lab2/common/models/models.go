package models

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

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
	Hash          string    `bson:"hash"`
	SubTaskNumber int       `bson:"subTaskNumber"`
	Status        string    `bson:"status"` // RECEIVED, PUBLISHED, WORKING, COMPLETE
	CreatedAt     time.Time `bson:"createdAt"`
	UpdatedAt     time.Time `bson:"updatedAt"`
}

type TaskMessage struct {
	Hash          string `json:"hash"`
	MaxLength     int    `json:"maxLength"`
	SubTaskNumber int    `json:"subTaskNumber"`
	SubTaskCount  int    `json:"subTaskCount"`
}

type ResultMessage struct {
	Hash          string `json:"hash"`
	SubTaskNumber int    `json:"subTaskNumber"`
	Result        string `json:"result"`
}

func BsonFilterReceived() bson.M {
	return bson.M{"subTasks.status": "RECEIVED"}
}

func MarshalTaskMessage(msg TaskMessage) ([]byte, error) {
	return json.Marshal(msg)
}

func UnmarshalResultMessage(data []byte, res *ResultMessage) error {
	return json.Unmarshal(data, res)
}
