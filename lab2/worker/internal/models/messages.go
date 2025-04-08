package models

// TaskMessage описывает сообщение, получаемое из очереди "tasks" для обработки.
type TaskMessage struct {
	Hash          string `json:"hash"`
	MaxLength     int    `json:"maxLength"`
	SubTaskNumber int    `json:"subTaskNumber"`
	SubTaskCount  int    `json:"subTaskCount"`
}

// ResultMessage описывает сообщение результата, отправляемое в очередь "results".
type ResultMessage struct {
	Hash          string `json:"hash"`
	SubTaskNumber int    `json:"subTaskNumber"`
	WorkerId      string `json:"workerId"`
	Result        string `json:"result"`
}
