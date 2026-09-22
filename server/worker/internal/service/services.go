package service

import "github.com/imrishabk/chimera/services/worker/internal/repo"

type Services struct {
	Auth      AuthService
	Chat      ChatService
	RAG       RAGService
	IngestJob IngestJobService
}

func NewServices(repos *repo.Repositories) *Services {
	s := &Services{
		Auth: NewAuthService(repos.User, repos.UserSession),
		Chat: NewChatService(repos.User, repos.Session),
	}
	return s
}

func NewServicesWithRAG(repos *repo.Repositories, rag RAGService) *Services {
	s := NewServices(repos)
	s.RAG = rag
	s.IngestJob = NewIngestJobService(repos.IngestJob, rag)
	return s
}
