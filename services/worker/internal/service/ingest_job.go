package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/imrishabk/chimera/services/worker/internal/repo"
)

type IngestJobService interface {
	CreateJob(ctx context.Context, req *model.IngestRequest, userID uuid.UUID) (*model.IngestJob, error)
	GetJob(ctx context.Context, id uuid.UUID) (*model.IngestJob, error)
	ListJobs(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]model.IngestJob, error)
}

type ingestJobService struct {
	jobs repo.IngestJobRepository
	rag  RAGService
}

func NewIngestJobService(jobs repo.IngestJobRepository, rag RAGService) IngestJobService {
	return &ingestJobService{jobs: jobs, rag: rag}
}

func (s *ingestJobService) CreateJob(ctx context.Context, req *model.IngestRequest, userID uuid.UUID) (*model.IngestJob, error) {
	source := req.Source
	if source == "" && len(req.Documents) > 0 {
		source = req.Documents[0].Source
	}
	sourceType := ""
	if len(req.Documents) > 0 {
		sourceType = req.Documents[0].SourceType
	}
	job := &model.IngestJob{
		SessionID:  req.SessionID,
		UserID:     userID,
		Status:     "pending",
		Source:     source,
		SourceType: sourceType,
		DocCount:   len(req.Documents),
	}
	created, err := s.jobs.Create(ctx, job)
	if err != nil {
		return nil, err
	}
	// mark processing
	if _, err := s.jobs.UpdateStatus(ctx, created.ID, "processing", created.DocCount, ""); err != nil {
		return nil, err
	}
	if s.rag == nil {
		failed, _ := s.jobs.UpdateStatus(ctx, created.ID, "failed", 0, "AI Core unavailable — gRPC client not connected")
		return failed, nil
	}
	resp, err := s.rag.IngestDocuments(ctx, req)
	if err != nil {
		failed, _ := s.jobs.UpdateStatus(ctx, created.ID, "failed", 0, err.Error())
		return failed, err
	}
	return s.jobs.UpdateStatus(ctx, created.ID, "completed", int(resp.Count), resp.Error)
}

func (s *ingestJobService) GetJob(ctx context.Context, id uuid.UUID) (*model.IngestJob, error) {
	return s.jobs.Get(ctx, id)
}

func (s *ingestJobService) ListJobs(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]model.IngestJob, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.jobs.ListBySession(ctx, sessionID, limit, offset)
}
