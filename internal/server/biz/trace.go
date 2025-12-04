package biz

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/trace"
)

type TraceServiceParams struct {
	fx.In
	RequestService *RequestService
	Ent            *ent.Client
}

func NewTraceService(params TraceServiceParams) *TraceService {
	return &TraceService{
		AbstractService: &AbstractService{
			db: params.Ent,
		},
		requestService: params.RequestService,
	}
}

type TraceService struct {
	*AbstractService
	requestService *RequestService
}

// CreateTrace creates a new trace record
func (s *TraceService) CreateTrace(ctx context.Context, httpReq *http.Request) (*ent.Trace, error) {
	// Get project ID from context (default to 1 if not set)
	projectID := 1
	if projectIDFromCtx, ok := contexts.GetProjectID(ctx); ok {
		projectID = projectIDFromCtx
	}

	// Get trace ID from context
	traceID, ok := contexts.GetTraceID(ctx)
	if !ok {
		return nil, fmt.Errorf("trace ID not found in context")
	}

	trace, err := s.entFromContext(ctx).Trace.Create().
		SetProjectID(projectID).
		SetTraceID(traceID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace: %w", err)
	}

	return trace, nil
}

// UpdateTraceFromRequest updates trace with request information
func (s *TraceService) UpdateTraceFromRequest(ctx context.Context, traceID int, httpReq *http.Request) error {
	// Since Trace schema doesn't have request URL field, this method just validates the trace exists
	_, err := s.entFromContext(ctx).Trace.Get(ctx, traceID)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("trace not found: %d", traceID)
		}
		return fmt.Errorf("failed to get trace: %w", err)
	}

	// Trace schema doesn't have request URL field, so nothing to update
	return nil
}

// UpdateTraceFromResponse updates trace with response information
func (s *TraceService) UpdateTraceFromResponse(ctx context.Context, traceID int, httpResp *http.Response) error {
	// Since Trace schema doesn't have response fields, this method just validates the trace exists
	_, err := s.entFromContext(ctx).Trace.Get(ctx, traceID)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("trace not found: %d", traceID)
		}
		return fmt.Errorf("failed to get trace: %w", err)
	}

	// Trace schema doesn't have response status or response body fields, so nothing to update
	return nil
}

// GetThreadFirstTrace gets the first trace for a thread
func (s *TraceService) GetThreadFirstTrace(ctx context.Context, threadID int) (*ent.Trace, error) {
	client := s.entFromContext(ctx)
	trace, err := client.Trace.Query().
		Where(trace.ThreadIDEQ(threadID)).
		Order(ent.Asc(trace.FieldCreatedAt)).
		First(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get first trace for thread: %w", err)
	}

	return trace, nil
}

// GetFirstUserQuery extracts the first user query from trace
func (s *TraceService) GetFirstUserQuery(ctx context.Context, traceID int) (*string, error) {
	// Since Trace schema doesn't have request data, return nil
	return nil, nil
}

// FirstText extracts the first text from trace
func (s *TraceService) FirstText(ctx context.Context, traceID int) (*string, error) {
	// Since Trace schema doesn't have request data, return nil
	return nil, nil
}

// GetRootSegment gets the root segment for a trace
func (s *TraceService) GetRootSegment(ctx context.Context, traceID int) (*Segment, error) {
	// Placeholder implementation
	return &Segment{ID: traceID}, nil
}

// FirstUserQuery extracts the first user query from trace
func (s *TraceService) FirstUserQuery(ctx context.Context, traceID int) (*string, error) {
	// Since Trace schema doesn't have request data, return nil
	return nil, nil
}

// GetOrCreateTrace gets an existing trace or creates a new one
func (s *TraceService) GetOrCreateTrace(ctx context.Context, projectID int, traceIDStr string, threadID *int) (*ent.Trace, error) {
	// Get trace ID from context
	traceIDFromCtx, ok := contexts.GetTraceID(ctx)
	if !ok {
		return nil, fmt.Errorf("trace ID not found in context")
	}

	traceID := traceIDFromCtx

	
	client := s.entFromContext(ctx)

	// Try to find existing trace
	trace, err := client.Trace.Query().
		Where(trace.TraceIDEQ(traceID)).
		Only(ctx)
	if err == nil {
		return trace, nil
	}

	// If not found and error is not "not found", return error
	if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to query trace: %w", err)
	}

	// Create new trace if not found
	return s.CreateTrace(ctx, nil)
}