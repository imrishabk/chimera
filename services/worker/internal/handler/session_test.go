package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	aicorepb "github.com/imrishabk/chimera/services/worker/internal/grpc"
	"github.com/imrishabk/chimera/services/worker/internal/model"
)

type mockChatService struct {
	createSession func(ctx context.Context, userID uuid.UUID) (*model.Session, error)
	getSession    func(ctx context.Context, id uuid.UUID) (*model.Session, error)
	listSessions  func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Session, error)
	createChat    func(ctx context.Context, req *model.ChatRequest) (*aicorepb.ChatResponse, error)
	listChats     func(ctx context.Context, sid uuid.UUID) ([]*aicorepb.Message, error)
}

func (m *mockChatService) CreateSession(ctx context.Context, userID uuid.UUID) (*model.Session, error) {
	if m.createSession != nil {
		return m.createSession(ctx, userID)
	}
	return &model.Session{ID: uuid.New(), UserID: userID}, nil
}

func (m *mockChatService) GetSession(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	if m.getSession != nil {
		return m.getSession(ctx, id)
	}
	return &model.Session{ID: id}, nil
}

func (m *mockChatService) ListSessions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.Session, error) {
	if m.listSessions != nil {
		return m.listSessions(ctx, userID, limit, offset)
	}
	return []model.Session{}, nil
}

func (m *mockChatService) CreateChat(ctx context.Context, req *model.ChatRequest) (*aicorepb.ChatResponse, error) {
	if m.createChat != nil {
		return m.createChat(ctx, req)
	}
	return &aicorepb.ChatResponse{SessionId: req.SessionID.String()}, nil
}

func (m *mockChatService) ListChats(ctx context.Context, sid uuid.UUID) ([]*aicorepb.Message, error) {
	if m.listChats != nil {
		return m.listChats(ctx, sid)
	}
	return []*aicorepb.Message{}, nil
}

func TestSessionCreateInvalidJSON(t *testing.T) {
	h := NewSessionHandler(&mockChatService{})
	req := httptest.NewRequest(http.MethodPost, "/session", strings.NewReader("{invalid"))
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 got %d", w.Code)
	}
}

func TestSessionCreateMissingUserID(t *testing.T) {
	h := NewSessionHandler(&mockChatService{})
	req := httptest.NewRequest(http.MethodPost, "/session", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.Create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing user_id, got %d", w.Code)
	}
}

func TestSessionGetInvalidID(t *testing.T) {
	h := NewSessionHandler(&mockChatService{})
	req := httptest.NewRequest(http.MethodGet, "/session/invalid", nil)
	w := httptest.NewRecorder()
	h.Get(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 got %d", w.Code)
	}
}
