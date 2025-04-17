package models

import (
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// HashTask представляет собой задачу расшифровки определенного хеша, которая может быть разделена на подзадачи.
type HashTask struct {
	RequestId          string    `bson:"requestId"`
	Hash               string    `bson:"hash"`
	MaxLength          int       `bson:"maxLength"`
	Status             string    `bson:"status"` // например "IN_PROGRESS", "DONE", "FAIL"
	SubTaskCount       int       `bson:"subTaskCount"`
	CompletedTaskCount int       `bson:"completedTaskCount"`
	Result             string    `bson:"result,omitempty"` // расшифрованный пароль, если найден
	SubTasks           []SubTask `bson:"subTasks"`
	CreatedAt          time.Time `bson:"createdAt"`
	// UpdatedAt отсутствует на уровне задачи (каждая SubTask имеет свой UpdatedAt)
}

// SubTask представляет часть пространства поиска для задачи HashTask.
type SubTask struct {
	Hash          string    `bson:"hash"`
	SubTaskNumber int       `bson:"subTaskNumber"`
	Status        string    `bson:"status"` // например "RECEIVED", "PUBLISHED, "COMPLETE"
	CreatedAt     time.Time `bson:"createdAt"`
	UpdatedAt     time.Time `bson:"updatedAt"`
}

// TaskMessage - структура сообщения, отправляемого воркерам через очередь "tasks".
type TaskMessage struct {
	Hash          string `json:"hash"`
	MaxLength     int    `json:"maxLength"`
	SubTaskNumber int    `json:"subTaskNumber"`
	SubTaskCount  int    `json:"subTaskCount"`
}

// ResultMessage - структура сообщения, отправляемого обратно через очередь "results".
type ResultMessage struct {
	Hash          string `json:"hash"`
	SubTaskNumber int    `json:"subTaskNumber"`
	Result        string `json:"result"`
}

// BsonFilterReceived возвращает фильтр MongoDB для поиска задач с подзадачами в статусе "RECEIVED".
func BsonFilterReceived() bson.M {
	return bson.M{"subTasks.status": "RECEIVED"}
}

// MarshalTaskMessage сериализует TaskMessage в JSON.
func MarshalTaskMessage(msg TaskMessage) ([]byte, error) {
	return json.Marshal(msg)
}

// UnmarshalResultMessage десериализует JSON-данные в структуру ResultMessage.
func UnmarshalResultMessage(data []byte, res *ResultMessage) error {
	return json.Unmarshal(data, res)
}
