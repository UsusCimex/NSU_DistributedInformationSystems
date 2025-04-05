package models

type CrackTaskRequest struct {
	Hash       string `json:"hash"`
	MaxLength  int    `json:"maxLength"`
	PartNumber int    `json:"partNumber"`
	PartCount  int    `json:"partCount"`
}

type CrackTaskResult struct {
	Hash       string `json:"hash"`
	Result     string `json:"result"`
	PartNumber int    `json:"partNumber"`
}
