package grpcclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	aicorepb "github.com/imrishabk/chimera/services/worker/internal/grpc"
)

type Client struct {
	conn   *grpc.ClientConn
	client aicorepb.AIServiceClient
}

func NewClient(addr string) (*Client, error) {
	if addr == "" {
		addr = "localhost:50051"
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to AI Core gRPC: %w", err)
	}
	return &Client{
		conn:   conn,
		client: aicorepb.NewAIServiceClient(conn),
	}, nil
}

func (c *Client) Chat(ctx context.Context, sessionID string, messages []*aicorepb.Message, temperature float32, maxTokens int32) (*aicorepb.ChatResponse, error) {
	req := &aicorepb.ChatRequest{
		SessionId:   sessionID,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
	return c.client.Chat(ctx, req)
}

func (c *Client) ChatStream(ctx context.Context, sessionID string, messages []*aicorepb.Message, temperature float32, maxTokens int32) (aicorepb.AIService_ChatStreamClient, error) {
	req := &aicorepb.ChatRequest{
		SessionId:   sessionID,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}
	return c.client.ChatStream(ctx, req)
}

func (c *Client) IngestDocuments(ctx context.Context, sessionID string, documents []*aicorepb.Document) (*aicorepb.IngestResponse, error) {
	req := &aicorepb.IngestRequest{
		SessionId: sessionID,
		Documents: documents,
	}
	return c.client.IngestDocuments(ctx, req)
}

func (c *Client) QueryRAG(ctx context.Context, sessionID, query string, k int32, filter map[string]string) (*aicorepb.QueryResponse, error) {
	req := &aicorepb.QueryRequest{
		SessionId: sessionID,
		Query:     query,
		K:         k,
		Filter:    filter,
	}
	return c.client.QueryRAG(ctx, req)
}

func (c *Client) Health(ctx context.Context) (*aicorepb.HealthResponse, error) {
	return c.client.Health(ctx, &aicorepb.HealthRequest{})
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
