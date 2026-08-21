package model

import aicorepb "github.com/imrishabk/chimera/services/worker/internal/grpc"

type ResponseData interface {
	User | Session | aicorepb.ChatResponse |
		aicorepb.IngestResponse | aicorepb.QueryResponse | aicorepb.HealthResponse
}

type Response[T User | Session] struct {
	Success bool   `json:"success"`
	Data    *T     `json:"data"`
	Error   string `json:"error"`
}
