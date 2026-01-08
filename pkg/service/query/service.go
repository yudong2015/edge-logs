package query

import (
	"context"

	"github.com/outpostos/edge-logs/pkg/model/request"
	"github.com/outpostos/edge-logs/pkg/model/response"
	"github.com/outpostos/edge-logs/pkg/repository/clickhouse"
)

// Service provides log query business logic
type Service struct {
	repo clickhouse.Repository
}

// NewService creates a new query service
func NewService(repo clickhouse.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// QueryLogs queries logs with business logic
func (s *Service) QueryLogs(ctx context.Context, req *request.LogQueryRequest) (*response.LogQueryResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Query from repository
	logs, total, err := s.repo.QueryLogs(ctx, req)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	responseLogs := make([]response.LogEntry, 0, len(logs))
	for _, log := range logs {
		responseLogs = append(responseLogs, response.LogEntry{
			ID:        log.ID,
			Timestamp: log.Timestamp,
			Message:   log.Message,
			Level:     log.Level,
			Namespace: log.Namespace,
			Pod:       log.Pod,
			Container: log.Container,
			Labels:    log.Labels,
		})
	}

	return &response.LogQueryResponse{
		Logs:       responseLogs,
		TotalCount: total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		HasMore:    len(responseLogs) == req.PageSize,
	}, nil
}