package models

type HashCrackRequest struct {
	Hash      string `json:"hash"`
	MaxLength int    `json:"maxLength"`
}

type HashCrackResponse struct {
	RequestId string `json:"requestId"`
}
