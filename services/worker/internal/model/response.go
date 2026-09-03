package model

type APIResponse[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

type APIErrorResponse struct {
	Success bool     `json:"success"`
	Error   []string `json:"error"`
}
