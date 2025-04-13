package models

// TaskMessage описывает задачу из очереди "tasks".
type TaskMessage struct {
	Hash          string `json:"hash"`
	MaxLength     int    `json:"maxLength"`
	SubTaskNumber int    `json:"subTaskNumber"`
	SubTaskCount  int    `json:"subTaskCount"`
}

// ResultMessage описывает сообщение результата для очереди "results".
type ResultMessage struct {
	Hash          string `json:"hash"`
	SubTaskNumber int    `json:"subTaskNumber"`
	Result        string `json:"result"`
}
