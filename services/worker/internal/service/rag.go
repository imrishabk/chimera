package service

import (
	"context"
	"fmt"

	aicorepb "github.com/imrishabk/chimera/services/worker/internal/grpc"
	grpcclient "github.com/imrishabk/chimera/services/worker/internal/grpc/client"
	"github.com/imrishabk/chimera/services/worker/internal/model"
)

type RAGService interface {
	IngestDocuments(ctx context.Context, req *model.IngestRequest) (*aicorepb.IngestResponse, error)
	QueryRAG(ctx context.Context, req *model.QueryRequest) (*aicorepb.QueryResponse, error)
}

type ragService struct {
	client *grpcclient.Client
}

func NewRAGService(client *grpcclient.Client) RAGService {
	return &ragService{client: client}
}

func (s *ragService) IngestDocuments(ctx context.Context, req *model.IngestRequest) (*aicorepb.IngestResponse, error) {
	if req == nil || req.SessionID.String() == "00000000-0000-0000-0000-000000000000" {
		return nil, fmt.Errorf("session_id required")
	}
	if s.client == nil {
		return nil, fmt.Errorf("AI Core unavailable — gRPC client not connected")
	}
	docs := make([]*aicorepb.Document, 0, len(req.Documents))
	for i, d := range req.Documents {
		docs = append(docs, &aicorepb.Document{
			Content:    d.Content,
			Source:     d.Source,
			SourceType: d.SourceType,
			DocId:      d.DocID,
			ChunkIndex: d.ChunkIndex,
			PageTitle:  d.PageTitle,
			Chapter:    d.Chapter,
			Metadata:   d.Metadata,
		})
		// ensure chunk index defaults to slice position if not set
		if d.ChunkIndex == 0 && len(req.Documents) > 1 {
			docs[i].ChunkIndex = int32(i)
		}
	}
	if len(docs) == 0 && req.Content != "" {
		docs = append(docs, &aicorepb.Document{
			Content: req.Content,
			Source:  req.Source,
		})
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no documents to ingest")
	}
	// batch to avoid oversized gRPC message (50 docs per batch)
	const batchSize = 50
	var finalResp *aicorepb.IngestResponse
	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}
		resp, err := s.client.IngestDocuments(ctx, req.SessionID.String(), docs[i:end])
		if err != nil {
			return nil, err
		}
		if finalResp == nil {
			finalResp = resp
		} else {
			finalResp.Count += resp.Count
			finalResp.DocsId = append(finalResp.DocsId, resp.DocsId...)
			if resp.Error != "" {
				finalResp.Error = resp.Error
			}
		}
	}
	return finalResp, nil
}

func (s *ragService) QueryRAG(ctx context.Context, req *model.QueryRequest) (*aicorepb.QueryResponse, error) {
	if req == nil || req.Query == "" {
		return nil, fmt.Errorf("query required")
	}
	if s.client == nil {
		return nil, fmt.Errorf("AI Core unavailable — gRPC client not connected")
	}
	k := req.K
	if k == 0 {
		k = 4
	}
	return s.client.QueryRAG(ctx, req.SessionID.String(), req.Query, k, req.Filter)
}
