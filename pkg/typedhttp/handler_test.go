package typedhttp_test

import (
	"context"
	"testing"

	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for handler testing
type TestRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

type TestResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
}

// Simple test handler implementation
type TestHandler struct {
	shouldFail bool
}

func (h *TestHandler) Handle(ctx context.Context, req TestRequest) (TestResponse, error) {
	if h.shouldFail {
		return TestResponse{}, typedhttp.NewValidationError("Test error", nil)
	}
	
	return TestResponse{
		Message: "Hello " + req.Name,
		ID:      "test-id",
	}, nil
}

func TestHandler_Interface(t *testing.T) {
	handler := &TestHandler{}
	
	// Verify that TestHandler implements Handler interface
	var _ typedhttp.Handler[TestRequest, TestResponse] = handler
	
	ctx := context.Background()
	req := TestRequest{
		Name:  "John",
		Email: "john@example.com",
	}
	
	resp, err := handler.Handle(ctx, req)
	
	require.NoError(t, err)
	assert.Equal(t, "Hello John", resp.Message)
	assert.Equal(t, "test-id", resp.ID)
}

func TestHandler_ErrorHandling(t *testing.T) {
	handler := &TestHandler{shouldFail: true}
	
	ctx := context.Background()
	req := TestRequest{
		Name:  "John",
		Email: "john@example.com",
	}
	
	_, err := handler.Handle(ctx, req)
	
	require.Error(t, err)
	
	var valErr *typedhttp.ValidationError
	assert.ErrorAs(t, err, &valErr)
	assert.Equal(t, "Test error", valErr.Message)
}

// Test service interface compatibility
type TestService struct{}

func (s *TestService) Execute(ctx context.Context, req TestRequest) (TestResponse, error) {
	return TestResponse{
		Message: "Service response",
		ID:      "service-id",
	}, nil
}

func TestService_Interface(t *testing.T) {
	service := &TestService{}
	
	// Verify that TestService implements Service interface
	var _ typedhttp.Service[TestRequest, TestResponse] = service
	
	ctx := context.Background()
	req := TestRequest{
		Name:  "Jane",
		Email: "jane@example.com",
	}
	
	resp, err := service.Execute(ctx, req)
	
	require.NoError(t, err)
	assert.Equal(t, "Service response", resp.Message)
	assert.Equal(t, "service-id", resp.ID)
}

func TestNewHTTPHandler(t *testing.T) {
	handler := &TestHandler{}
	
	httpHandler := typedhttp.NewHTTPHandler(handler)
	
	assert.NotNil(t, httpHandler)
	
	// Test with options
	httpHandlerWithOpts := typedhttp.NewHTTPHandler(
		handler,
		typedhttp.WithTags("test"),
		typedhttp.WithSummary("Test handler"),
		typedhttp.WithDefaultObservability(),
	)
	
	assert.NotNil(t, httpHandlerWithOpts)
}