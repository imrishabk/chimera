package service

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	aicorepb "github.com/imrishabk/chimera/services/worker/internal/grpc"
	grpcclient "github.com/imrishabk/chimera/services/worker/internal/grpc/client"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/imrishabk/chimera/services/worker/internal/repo"
)

type ChatService interface {
	CreateSession(c context.Context, userID uuid.UUID) (*model.Session, error)
	GetSession(c context.Context, sessionID uuid.UUID) (*model.Session, error)
	ListSessions(c context.Context, userID uuid.UUID, limit, offset int) ([]model.Session, error)
	CreateChat(c context.Context, r *model.ChatRequest) (*aicorepb.ChatResponse, error)
	ListChats(c context.Context, sessionID uuid.UUID) ([]*aicorepb.Message, error)
}

type chatService struct {
	user       repo.UserRepository
	session    repo.SessionRepository
	grpcClient *grpcclient.Client
}

func NewChatService(userRepo repo.UserRepository, sessionRepo repo.SessionRepository) ChatService {
	grpcAddr := os.Getenv("GRPC_AI_HOST")
	if grpcAddr == "" {
		grpcAddr = "localhost:50051"
	}
	client, err := grpcclient.NewClient(grpcAddr)
	if err != nil {
		// If Python AI Core is not running, log but don't crash.
		// The service will fall back to stub responses.
		client = nil
		_ = err // suppressed for startup resilience
	}
	return &chatService{
		user:       userRepo,
		session:    sessionRepo,
		grpcClient: client,
	}
}

func (s *chatService) CreateSession(c context.Context, userID uuid.UUID) (*model.Session, error) {
	ses, err := s.session.CreateSession(c, userID)
	if err != nil {
		return nil, err
	}
	return ses, nil
}

func (s *chatService) GetSession(c context.Context, sessionID uuid.UUID) (*model.Session, error) {
	return s.session.FetchSession(c, sessionID)
}

func (s *chatService) ListSessions(c context.Context, userID uuid.UUID, limit, offset int) ([]model.Session, error) {
	return s.session.ListSessionByUserID(c, userID, limit, offset)
}

func (s *chatService) ListChats(c context.Context, sessionID uuid.UUID) ([]*aicorepb.Message, error) {
	// Chat history is stored in Python AI Core via PostgresChatMessageHistory.
	// For now return empty; when QueryRAG/Chat history fetch is needed, add gRPC method.
	_ = c
	_ = sessionID
	return []*aicorepb.Message{}, nil
}

func (s *chatService) CreateChat(c context.Context, r *model.ChatRequest) (*aicorepb.ChatResponse, error) {
	if r == nil || r.SessionID == uuid.Nil {
		return nil, fmt.Errorf("chat request or session_id is required")
	}

	if s.grpcClient == nil {
		return &aicorepb.ChatResponse{
			SessionId: r.SessionID.String(),
			Message: &aicorepb.Message{
				Role:    "assistant",
				Content: "AI Core unavailable — gRPC client not connected to Python service at localhost:50051",
			},
			Done: true,
		}, nil
	}

	// Convert model.ChatRequest to gRPC proto messages
	protoMessages := []*aicorepb.Message{
		{
			Role:    "user",
			Content: r.Text,
		},
	}

	resp, err := s.grpcClient.Chat(c, r.SessionID.String(), protoMessages, 0.7, 0)
	if err != nil {
		return nil, fmt.Errorf("AI Core Chat call failed: %w", err)
	}

	return resp, nil
}
