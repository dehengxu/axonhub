package biz

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/request"
	"github.com/looplj/axonhub/internal/ent/requestexecution"
	"github.com/looplj/axonhub/internal/objects"
)

type RequestServiceParams struct {
	fx.In
	Ent *ent.Client
}

func NewRequestService(params RequestServiceParams) *RequestService {
	return &RequestService{
		AbstractService: &AbstractService{
			db: params.Ent,
		},
	}
}

type RequestService struct {
	*AbstractService
}

// CreateRequest creates a new request record
func (s *RequestService) CreateRequest(ctx context.Context, apiKeyID *int, projectID int, traceID *int, channelID *int, source string, modelID string, format string, requestBody interface{}) (*ent.Request, error) {
	client := s.entFromContext(ctx)
	createBuilder := client.Request.Create().
		SetProjectID(projectID).
		SetSource(request.Source(source)).
		SetModelID(modelID).
		SetFormat(format).
		SetRequestBody(objects.JSONRawMessage(fmt.Sprintf("%v", requestBody)))

	if apiKeyID != nil {
		createBuilder = createBuilder.SetAPIKeyID(*apiKeyID)
	}
	if traceID != nil {
		createBuilder = createBuilder.SetTraceID(*traceID)
	}
	if channelID != nil {
		createBuilder = createBuilder.SetChannelID(*channelID)
	}

	request, err := createBuilder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return request, nil
}

// UpdateRequestCompleted updates request as completed
func (s *RequestService) UpdateRequestCompleted(ctx context.Context, requestID int, executionID int, responseBody interface{}) error {
	client := s.entFromContext(ctx)
	_, err := client.Request.UpdateOneID(requestID).
		SetStatus(request.StatusCompleted).
		SetResponseBody(objects.JSONRawMessage(fmt.Sprintf("%v", responseBody))).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update request completed: %w", err)
	}
	return nil
}

// UpdateRequestStatusFromError updates request status from error
func (s *RequestService) UpdateRequestStatusFromError(ctx context.Context, requestID int, err error) error {
	client := s.entFromContext(ctx)
	status := request.StatusFailed
	if err != nil {
		status = request.StatusFailed
	}

	_, updateErr := client.Request.UpdateOneID(requestID).
		SetStatus(status).
		Save(ctx)
	if updateErr != nil {
		return fmt.Errorf("failed to update request status from error: %w", updateErr)
	}
	return nil
}

// CreateRequestExecution creates a new request execution record
func (s *RequestService) CreateRequestExecution(ctx context.Context, requestID int, channelID int, requestDir string, requestBody string) (*ent.RequestExecution, error) {
	client := s.entFromContext(ctx)
	createBuilder := client.RequestExecution.Create().
		SetRequestID(requestID).
		SetChannelID(channelID)

	if requestBody != "" {
		createBuilder = createBuilder.SetRequestBody(objects.JSONRawMessage(requestBody))
	}

	execution, err := createBuilder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create request execution: %w", err)
	}
	return execution, nil
}

// UpdateRequestExecutionCompleted updates request execution as completed
func (s *RequestService) UpdateRequestExecutionCompleted(ctx context.Context, executionID int, responseBody string, statusCode int64) error {
	client := s.entFromContext(ctx)
	_, err := client.RequestExecution.UpdateOneID(executionID).
		SetResponseBody(objects.JSONRawMessage(responseBody)).
		SetStatus(requestexecution.StatusCompleted).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update request execution completed: %w", err)
	}
	return nil
}

// UpdateRequestExecutionStatusFromError updates execution status from error
func (s *RequestService) UpdateRequestExecutionStatusFromError(ctx context.Context, executionID int, err error) error {
	client := s.entFromContext(ctx)
	status := requestexecution.StatusFailed
	if err != nil {
		status = requestexecution.StatusFailed
	}

	_, updateErr := client.RequestExecution.UpdateOneID(executionID).
		SetStatus(status).
		Save(ctx)
	if updateErr != nil {
		return fmt.Errorf("failed to update request execution status from error: %w", updateErr)
	}
	return nil
}

// UpdateRequestExecutionFailed updates request execution as failed
func (s *RequestService) UpdateRequestExecutionFailed(ctx context.Context, executionID int, statusCode int64, errorMessage string) error {
	client := s.entFromContext(ctx)
	updateBuilder := client.RequestExecution.UpdateOneID(executionID).
		SetStatus(requestexecution.StatusFailed)

	if errorMessage != "" {
		updateBuilder = updateBuilder.SetErrorMessage(errorMessage)
	}

	_, err := updateBuilder.Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update request execution failed: %w", err)
	}
	return nil
}

// AppendRequestExecutionChunk appends a response chunk to request execution
func (s *RequestService) AppendRequestExecutionChunk(ctx context.Context, executionID int, chunk []byte) error {
	// This is a placeholder implementation since RequestExecution schema doesn't have chunks field
	return nil
}

// UpdateRequestChannelID updates the channel ID for a request
func (s *RequestService) UpdateRequestChannelID(ctx context.Context, requestID int, channelID int) error {
	client := s.entFromContext(ctx)
	_, err := client.Request.UpdateOneID(requestID).
		SetChannelID(channelID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("failed to update request channel ID: %w", err)
	}
	return nil
}

// GetLastSuccessfulChannelID gets the last successful channel ID
func (s *RequestService) GetLastSuccessfulChannelID(ctx context.Context, requestID int) (int, error) {
	// Placeholder implementation
	return 0, nil
}

// AppendRequestChunk appends a chunk to request
func (s *RequestService) AppendRequestChunk(ctx context.Context, requestID int, chunk []byte) error {
	// Placeholder implementation since Request schema doesn't have chunks field in this context
	return nil
}

// LoadRequestBody loads request body from storage
func (s *RequestService) LoadRequestBody(ctx context.Context, request *ent.Request) (string, error) {
	// Placeholder implementation
	return "", nil
}

// LoadResponseBody loads response body from storage
func (s *RequestService) LoadResponseBody(ctx context.Context, request *ent.Request) (string, error) {
	// Placeholder implementation
	return "", nil
}

// LoadResponseChunks loads response chunks from storage
func (s *RequestService) LoadResponseChunks(ctx context.Context, request *ent.Request) ([]string, error) {
	// Placeholder implementation
	return nil, nil
}

// LoadRequestExecutionRequestBody loads request execution request body from storage
func (s *RequestService) LoadRequestExecutionRequestBody(ctx context.Context, execution *ent.RequestExecution) (string, error) {
	// Placeholder implementation
	return "", nil
}

// LoadRequestExecutionResponseBody loads request execution response body from storage
func (s *RequestService) LoadRequestExecutionResponseBody(ctx context.Context, execution *ent.RequestExecution) (string, error) {
	// Placeholder implementation
	return "", nil
}

// LoadRequestExecutionResponseChunks loads request execution response chunks from storage
func (s *RequestService) LoadRequestExecutionResponseChunks(ctx context.Context, execution *ent.RequestExecution) ([]string, error) {
	// Placeholder implementation
	return nil, nil
}

// Key generation functions for storage paths
func GenerateRequestBodyKey(projectID, requestID int) string {
	return fmt.Sprintf("requests/%d/request_%d_body.json", projectID, requestID)
}

func GenerateResponseBodyKey(projectID, requestID int) string {
	return fmt.Sprintf("requests/%d/request_%d_response.json", projectID, requestID)
}

func GenerateResponseChunksKey(projectID, requestID int) string {
	return fmt.Sprintf("requests/%d/request_%d_chunks", projectID, requestID)
}

func GenerateRequestExecutionsDirKey(projectID, requestID int) string {
	return fmt.Sprintf("requests/%d/request_%d_executions", projectID, requestID)
}

func GenerateRequestDirKey(projectID, requestID int) string {
	return fmt.Sprintf("requests/%d/request_%d", projectID, requestID)
}

func GenerateExecutionRequestBodyKey(projectID, requestID, executionID int) string {
	return fmt.Sprintf("requests/%d/request_%d/execution_%d_body.json", projectID, requestID, executionID)
}

func GenerateExecutionResponseBodyKey(projectID, requestID, executionID int) string {
	return fmt.Sprintf("requests/%d/request_%d/execution_%d_response.json", projectID, requestID, executionID)
}

func GenerateExecutionResponseChunksKey(projectID, requestID, executionID int) string {
	return fmt.Sprintf("requests/%d/request_%d/execution_%d_chunks", projectID, requestID, executionID)
}

func GenerateExecutionRequestDirKey(projectID, requestID, executionID int) string {
	return fmt.Sprintf("requests/%d/request_%d/execution_%d", projectID, requestID, executionID)
}