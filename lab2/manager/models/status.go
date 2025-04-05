package models

type StatusResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}
