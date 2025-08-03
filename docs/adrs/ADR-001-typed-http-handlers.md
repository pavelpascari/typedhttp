# ADR-004: Typed HTTP Handler Abstraction

## Status

**Accepted** - Implemented ✅
**Implementation Date**: July 2025

## Executive Summary

This ADR proposes a typed HTTP handler abstraction using Go generics to provide compile-time type safety, reduce boilerplate, and enable automatic OpenAPI generation while maintaining clean architectural separation. The solution addresses current pain points around manual JSON handling, runtime errors, and documentation drift, but introduces complexity that requires careful implementation and team training.

## Context

Our current HTTP handler implementation follows standard Go patterns with `http.HandlerFunc` signatures, requiring manual JSON marshalling/unmarshalling and lacking compile-time type safety. As our API grows, we face several challenges:

1. **Type Safety**: Current handlers use `interface{}` for request/response data, leading to potential runtime errors
2. **Code Duplication**: Each handler manually implements JSON parsing, validation, and response envelope wrapping
3. **OpenAPI Drift**: Manual maintenance of OpenAPI specifications leads to documentation-implementation mismatches
4. **Testing Complexity**: Testing requires complex HTTP request/response mocking
5. **Business Logic Coupling**: HTTP concerns are mixed with business logic, violating hexagonal architecture principles

### Current Handler Pattern

```go
func (h *TransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
    var req models.CreateTransactionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSONError(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    transaction, err := h.repo.Create(&models.Transaction{
        Amount:      req.Amount,
        Description: req.Description,
        // ... manual mapping
    })
    
    if err != nil {
        writeJSONError(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    writeJSONResponse(w, transaction, http.StatusCreated)
}
```

### Success Criteria

1. **Idiomatic Go**: Must follow Go conventions and be easily extensible
2. **Zero Runtime Overhead**: Should use compile-time techniques (generics, interfaces)
3. **Hexagonal Architecture**: Support multiple adapters with clean separation of concerns
4. **OpenAPI Generation**: Automatically generate specifications from handler type annotations
5. **Built-in Observability**: Capture tracing and metrics before handler execution
6. **Composable Middleware**: Support both power users and simple getting-started scenarios

## Decision

We will implement a **Typed HTTP Handler Abstraction** using Go generics to provide compile-time type safety while maintaining clean architectural separation and enabling automatic OpenAPI generation.

### Core Architecture

#### 1. Handler Interface Hierarchy

```go
// Core business logic interface (transport-agnostic)
type Handler[TRequest, TResponse any] interface {
    Handle(ctx context.Context, req TRequest) (TResponse, error)
}

// Service interface for business logic layer (more idiomatic Go term)
type Service[TRequest, TResponse any] interface {
    Execute(ctx context.Context, req TRequest) (TResponse, error)
}

// HTTP-specific wrapper with flexible encoding/decoding
type HTTPHandler[TRequest, TResponse any] struct {
    handler    Handler[TRequest, TResponse]
    decoder    RequestDecoder[TRequest]
    encoder    ResponseEncoder[TResponse]
    middleware []Middleware
    metadata   OpenAPIMetadata
    errorMapper ErrorMapper
}

// Flexible encoding/decoding interfaces
type RequestDecoder[T any] interface {
    Decode(r *http.Request) (T, error)
    ContentTypes() []string
}

type ResponseEncoder[T any] interface {
    Encode(w http.ResponseWriter, data T, statusCode int) error
    ContentType() string
}

// Error mapping for different error types
type ErrorMapper interface {
    MapError(err error) (statusCode int, response interface{})
}
```

#### 2. Typed Router Integration

```go
type TypedRouter struct {
    *TrackedRouter
    schemas  map[reflect.Type]Schema
    handlers []HandlerRegistration
}

func (r *TypedRouter) RegisterHandler[TReq, TResp any](
    method, path string,
    handler Handler[TReq, TResp],
    opts ...HandlerOption,
) {
    // Create HTTP adapter
    httpHandler := r.createHTTPHandler(handler, opts...)
    
    // Register with TrackedRouter for contract validation
    r.TrackedRouter.MethodFunc(method, path, httpHandler)
    
    // Store metadata for OpenAPI generation
    r.registerHandlerMetadata[TReq, TResp](method, path, opts...)
}

// Convenience methods
func (r *TypedRouter) GET[TReq, TResp any](path string, handler Handler[TReq, TResp], opts ...HandlerOption) {
    r.RegisterHandler("GET", path, handler, opts...)
}

func (r *TypedRouter) POST[TReq, TResp any](path string, handler Handler[TReq, TResp], opts ...HandlerOption) {
    r.RegisterHandler("POST", path, handler, opts...)
}
```

#### 3. OpenAPI Metadata System

```go
type OpenAPIMetadata struct {
    Summary     string                     `json:"summary"`
    Description string                     `json:"description"`
    Tags        []string                   `json:"tags"`
    Parameters  []ParameterSpec            `json:"parameters,omitempty"`
    RequestBody *RequestBodySpec           `json:"requestBody,omitempty"`
    Responses   map[string]ResponseSpec    `json:"responses"`
}

type HandlerOption func(*HandlerConfig)

func WithOpenAPI(metadata OpenAPIMetadata) HandlerOption {
    return func(cfg *HandlerConfig) {
        cfg.Metadata = metadata
    }
}

func WithTags(tags ...string) HandlerOption {
    return func(cfg *HandlerConfig) {
        cfg.Metadata.Tags = tags
    }
}
```

#### 4. Observability Integration

```go
type ObservabilityConfig struct {
    Tracing         bool
    Metrics         bool
    Logging         bool
    TraceAttributes map[string]interface{}
    MetricLabels    map[string]string
}

func WithObservability(config ObservabilityConfig) HandlerOption {
    return func(cfg *HandlerConfig) {
        cfg.Observability = config
    }
}

func WithDefaultObservability() HandlerOption {
    return WithObservability(ObservabilityConfig{
        Tracing: true,
        Metrics: true,
        Logging: true,
    })
}
```

#### 5. Middleware Composition

```go
type Middleware func(http.Handler) http.Handler

func WithMiddleware(middleware ...Middleware) HandlerOption {
    return func(cfg *HandlerConfig) {
        cfg.Middleware = append(cfg.Middleware, middleware...)
    }
}

// Pre-defined middleware chains
func WithAuthentication() HandlerOption {
    return WithMiddleware(authMiddleware)
}

func WithRateLimit(limit int) HandlerOption {
    return WithMiddleware(createRateLimitMiddleware(limit))
}
```

### Usage Examples

#### 1. Simple Handler (Minimal Boilerplate)

```go
type GetTransactionsService struct {
    repo repository.TransactionRepository
}

type GetTransactionsRequest struct {
    FamilyID string `query:"family_id" validate:"required"`
    Limit    int    `query:"limit" validate:"min=1,max=100" default:"10"`
    Offset   int    `query:"offset" validate:"min=0" default:"0"`
}

type GetTransactionsResponse struct {
    Transactions []*models.Transaction `json:"transactions"`
    Total        int                   `json:"total"`
    HasMore      bool                  `json:"has_more"`
}

func (s *GetTransactionsService) Handle(ctx context.Context, req GetTransactionsRequest) (GetTransactionsResponse, error) {
    // Input validation happens automatically via struct tags
    transactions, total, err := s.repo.GetPaginated(ctx, req.FamilyID, req.Limit, req.Offset)
    if err != nil {
        // Different error types are handled by ErrorMapper
        if errors.Is(err, repository.ErrFamilyNotFound) {
            return GetTransactionsResponse{}, domain.NewNotFoundError("family", req.FamilyID)
        }
        return GetTransactionsResponse{}, fmt.Errorf("failed to fetch transactions: %w", err)
    }
    
    return GetTransactionsResponse{
        Transactions: transactions,
        Total:        total,
        HasMore:      req.Offset+req.Limit < total,
    }, nil
}

// Simple registration with defaults
r.GET("/api/v1/transactions", getTransactionsService)
```

#### 2. Error Handling Examples

```go
// Domain errors for business logic
type DomainError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Field   string `json:"field,omitempty"`
}

func (e *DomainError) Error() string {
    return e.Message
}

// Custom error mapper
type AppErrorMapper struct{}

func (m *AppErrorMapper) MapError(err error) (int, interface{}) {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        return http.StatusRequestTimeout, ErrorResponse{
            Error: "Request timeout",
            Code:  "TIMEOUT",
        }
    
    case errors.As(err, &domain.ValidationError{}):
        var valErr *domain.ValidationError
        errors.As(err, &valErr)
        return http.StatusBadRequest, ErrorResponse{
            Error: "Validation failed",
            Code:  "VALIDATION_ERROR",
            Details: valErr.Fields,
        }
    
    case errors.As(err, &domain.NotFoundError{}):
        var nfErr *domain.NotFoundError
        errors.As(err, &nfErr)
        return http.StatusNotFound, ErrorResponse{
            Error: fmt.Sprintf("%s not found", nfErr.Resource),
            Code:  "NOT_FOUND",
            ResourceID: nfErr.ID,
        }
    
    case errors.As(err, &domain.ConflictError{}):
        var conflictErr *domain.ConflictError
        errors.As(err, &conflictErr)
        return http.StatusConflict, ErrorResponse{
            Error: conflictErr.Message,
            Code:  "CONFLICT",
        }
    
    default:
        // Log internal errors but don't expose details
        logger.Error("Internal server error", "error", err)
        return http.StatusInternalServerError, ErrorResponse{
            Error: "Internal server error",
            Code:  "INTERNAL_ERROR",
        }
    }
}
```

#### 3. Multiple Format Support

```go
// JSON decoder (default)
type JSONDecoder[T any] struct{}

func (d *JSONDecoder[T]) Decode(r *http.Request) (T, error) {
    var result T
    if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
        return result, fmt.Errorf("invalid JSON: %w", err)
    }
    
    // Automatic validation
    if err := validator.Validate(result); err != nil {
        return result, domain.NewValidationError(err)
    }
    
    return result, nil
}

func (d *JSONDecoder[T]) ContentTypes() []string {
    return []string{"application/json"}
}

// XML decoder
type XMLDecoder[T any] struct{}

func (d *XMLDecoder[T]) Decode(r *http.Request) (T, error) {
    var result T
    if err := xml.NewDecoder(r.Body).Decode(&result); err != nil {
        return result, fmt.Errorf("invalid XML: %w", err)
    }
    return result, nil
}

func (d *XMLDecoder[T]) ContentTypes() []string {
    return []string{"application/xml", "text/xml"}
}

// Form decoder for simple requests
type FormDecoder[T any] struct{}

func (d *FormDecoder[T]) Decode(r *http.Request) (T, error) {
    var result T
    if err := r.ParseForm(); err != nil {
        return result, fmt.Errorf("invalid form data: %w", err)
    }
    
    // Use reflection or struct tags to map form values
    if err := mapFormToStruct(r.Form, &result); err != nil {
        return result, err
    }
    
    return result, nil
}

func (d *FormDecoder[T]) ContentTypes() []string {
    return []string{"application/x-www-form-urlencoded"}
}

// Multi-format encoder
type ContentNegotiationEncoder[T any] struct {
    jsonEncoder *JSONEncoder[T]
    xmlEncoder  *XMLEncoder[T]
}

func (e *ContentNegotiationEncoder[T]) Encode(w http.ResponseWriter, data T, statusCode int) error {
    accept := w.Header().Get("Accept")
    
    switch {
    case strings.Contains(accept, "application/xml"):
        return e.xmlEncoder.Encode(w, data, statusCode)
    default:
        return e.jsonEncoder.Encode(w, data, statusCode)
    }
}
```

#### 4. Advanced Handler (Full Control)

```go
type CreateTransactionService struct {
    repo   repository.TransactionRepository
    budget domain.BudgetService
    audit  domain.AuditService
}

type CreateTransactionRequest struct {
    FamilyID    string    `json:"family_id" validate:"required,cuid2"`
    Amount      float64   `json:"amount" validate:"required,gt=0"`
    Description string    `json:"description" validate:"required,min=1,max=500"`
    Category    string    `json:"category" validate:"required,oneof=groceries entertainment transport utilities"`
    Date        time.Time `json:"date" validate:"required"`
}

type CreateTransactionResponse struct {
    Transaction   *models.Transaction `json:"transaction"`
    BudgetImpact  *BudgetImpact      `json:"budget_impact"`
    Message       string             `json:"message"`
}

func (s *CreateTransactionService) Handle(ctx context.Context, req CreateTransactionRequest) (CreateTransactionResponse, error) {
    // Business logic with comprehensive error handling
    budget, err := s.budget.GetActiveBudget(ctx, req.FamilyID)
    if err != nil {
        return CreateTransactionResponse{}, fmt.Errorf("failed to get budget: %w", err)
    }
    
    // Check budget constraints
    if budget.WillExceedLimit(req.Category, req.Amount) {
        return CreateTransactionResponse{}, domain.NewConflictError(
            "Transaction would exceed budget limit for category %s", req.Category)
    }
    
    transaction := &models.Transaction{
        ID:          cuid2.Generate(),
        FamilyID:    req.FamilyID,
        Amount:      req.Amount,
        Description: req.Description,
        Category:    req.Category,
        Date:        req.Date,
        CreatedAt:   time.Now(),
    }
    
    savedTransaction, err := s.repo.Create(ctx, transaction)
    if err != nil {
        return CreateTransactionResponse{}, fmt.Errorf("failed to create transaction: %w", err)
    }
    
    // Calculate budget impact
    impact := budget.CalculateImpact(req.Category, req.Amount)
    
    // Audit trail
    s.audit.LogTransactionCreated(ctx, savedTransaction)
    
    return CreateTransactionResponse{
        Transaction:  savedTransaction,
        BudgetImpact: impact,
        Message:      "Transaction created successfully",
    }, nil
}

r.RegisterHandler("POST", "/api/v1/transactions", createTransactionService,
    WithOpenAPI(OpenAPIMetadata{
        Summary:     "Create a new transaction",
        Description: "Creates a new transaction within the specified family budget",
        Tags:        []string{"transactions", "budgeting"},
    }),
    WithObservability(ObservabilityConfig{
        Tracing: true,
        Metrics: true,
        TraceAttributes: map[string]interface{}{
            "operation": "create_transaction",
            "resource":  "transaction",
        },
    }),
    WithMultiFormat(), // Support JSON, XML, and form data
    WithCustomErrorMapper(&AppErrorMapper{}),
    WithAuthentication(),
    WithRateLimit(100),
    WithMiddleware(familyContextMiddleware),
)
```

#### 5. Handler Composition and Reusability

```go
// Base handler for common functionality
type BaseTransactionHandler struct {
    repo   repository.TransactionRepository
    logger *slog.Logger
}

// Composable handler components
type ValidationHandler[T any] struct {
    validator *validator.Validate
}

func (v *ValidationHandler[T]) ValidateRequest(ctx context.Context, req T) error {
    return v.validator.Struct(req)
}

type AuditHandler struct {
    auditService domain.AuditService
}

func (a *AuditHandler) LogOperation(ctx context.Context, operation string, data interface{}) {
    a.auditService.Log(ctx, operation, data)
}

// Composed handler using embedded components
type GetTransactionService struct {
    BaseTransactionHandler
    ValidationHandler[GetTransactionRequest]
    AuditHandler
}

func (s *GetTransactionService) Handle(ctx context.Context, req GetTransactionRequest) (GetTransactionResponse, error) {
    // Validation is handled automatically by the framework
    // but can also be called explicitly if needed
    if err := s.ValidateRequest(ctx, req); err != nil {
        return GetTransactionResponse{}, err
    }
    
    transaction, err := s.repo.GetByID(ctx, req.ID)
    if err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            return GetTransactionResponse{}, domain.NewNotFoundError("transaction", req.ID)
        }
        return GetTransactionResponse{}, fmt.Errorf("failed to get transaction: %w", err)
    }
    
    // Audit successful retrieval
    s.LogOperation(ctx, "transaction_retrieved", map[string]interface{}{
        "transaction_id": req.ID,
        "user_id":        getUserID(ctx),
    })
    
    return GetTransactionResponse{
        Transaction: transaction,
    }, nil
}

// Handler factory for consistent configuration
func NewTransactionHandlers(deps Dependencies) TransactionHandlers {
    base := BaseTransactionHandler{
        repo:   deps.TransactionRepo,
        logger: deps.Logger,
    }
    
    validator := ValidationHandler[any]{
        validator: deps.Validator,
    }
    
    audit := AuditHandler{
        auditService: deps.AuditService,
    }
    
    return TransactionHandlers{
        Get: &GetTransactionService{
            BaseTransactionHandler: base,
            ValidationHandler:      validator,
            AuditHandler:          audit,
        },
        Create: &CreateTransactionService{
            BaseTransactionHandler: base,
            ValidationHandler:      validator,
            AuditHandler:          audit,
            BudgetService:         deps.BudgetService,
        },
        // ... other handlers
    }
}

// Bulk registration with consistent configuration
func RegisterTransactionRoutes(r *TypedRouter, handlers TransactionHandlers) {
    opts := []HandlerOption{
        WithTags("transactions"),
        WithAuthentication(),
        WithDefaultObservability(),
        WithCustomErrorMapper(&AppErrorMapper{}),
    }
    
    r.GET("/api/v1/transactions/{id}", handlers.Get, opts...)
    r.POST("/api/v1/transactions", handlers.Create, opts...)
    r.PUT("/api/v1/transactions/{id}", handlers.Update, opts...)
    r.DELETE("/api/v1/transactions/{id}", handlers.Delete, opts...)
}
```

#### 6. Different Request/Response Patterns

```go
// Query parameter handling
type ListTransactionsRequest struct {
    FamilyID string `query:"family_id" validate:"required,cuid2"`
    Category string `query:"category" validate:"omitempty,oneof=groceries entertainment"`
    FromDate string `query:"from_date" validate:"omitempty,datetime=2006-01-02"`
    ToDate   string `query:"to_date" validate:"omitempty,datetime=2006-01-02"`
    Limit    int    `query:"limit" validate:"min=1,max=100" default:"10"`
    Offset   int    `query:"offset" validate:"min=0" default:"0"`
}

// Path parameter handling
type GetTransactionRequest struct {
    ID       string `path:"id" validate:"required,cuid2"`
    FamilyID string `header:"X-Family-ID" validate:"required,cuid2"`
}

// File upload handling
type ImportTransactionsRequest struct {
    FamilyID string                `form:"family_id" validate:"required,cuid2"`
    File     *multipart.FileHeader `form:"file" validate:"required"`
    Format   string                `form:"format" validate:"required,oneof=csv json xlsx"`
}

// Streaming response
type ExportTransactionsRequest struct {
    FamilyID string `query:"family_id" validate:"required,cuid2"`
    Format   string `query:"format" validate:"required,oneof=csv json xlsx pdf"`
}

type StreamingResponse struct {
    ContentType string
    Filename    string
    Stream      io.Reader
}
```

### OpenAPI Generation

The system automatically generates OpenAPI specifications by analyzing registered handler types:

```go
func (r *TypedRouter) GenerateOpenAPISpec() (*openapi3.T, error) {
    spec := &openapi3.T{
        OpenAPI: "3.0.3",
        Info: &openapi3.Info{
            Title:   "Budget App API",
            Version: "1.0.0",
        },
        Paths: openapi3.Paths{},
    }
    
    for _, handler := range r.handlers {
        pathItem := r.generatePathItem(handler)
        spec.Paths[handler.Path] = pathItem
    }
    
    // Add component schemas from registered types
    spec.Components = &openapi3.Components{
        Schemas: r.generateSchemas(),
    }
    
    return spec, nil
}
```

### Migration Strategy

#### Phase 1: Introduction (Backward Compatible)

- Introduce `TypedRouter` alongside existing `TrackedRouter`
- Create adapter for existing handlers
- Establish patterns and documentation

```go
// Existing handlers continue to work
r.Get("/legacy-endpoint", existingHandler)

// New handlers use typed approach
r.GET("/typed-endpoint", typedHandler)
```

#### Phase 2: Gradual Migration

- Migrate high-traffic endpoints first
- Update handler tests to use business logic directly
- Generate OpenAPI specs for migrated endpoints

#### Phase 3: Standardization

- Complete migration of all handlers
- Remove legacy pattern support (optional)
- Full OpenAPI spec generation

## Implementation Plan

### Core Components

1. **Handler Interfaces** (`internal/handlers/typed/interfaces.go`)
    - `Handler[TRequest, TResponse]` interface
    - `UseCase[TRequest, TResponse]` interface
    - Error handling patterns

2. **TypedRouter** (`internal/router/typed_router.go`)
    - Generic handler registration methods
    - Integration with existing TrackedRouter
    - Middleware chain composition

3. **OpenAPI Generator** (`internal/openapi/typed_generator.go`)
    - Type reflection for schema generation
    - Metadata extraction from handler options
    - Integration with existing OpenAPI validation

4. **Observability Integration** (`internal/o11y/typed_middleware.go`)
    - Automatic tracing span creation
    - Metrics collection for typed handlers
    - Structured logging integration

5. **Request/Response Codecs** (`internal/handlers/typed/codecs.go`)
    - JSON encoder/decoder with validation
    - Error response standardization
    - Envelope pattern integration

### Testing Strategy

#### Unit Testing (Business Logic)
```go
func TestCreateTransactionService_Handle(t *testing.T) {
    tests := []struct {
        name        string
        req         CreateTransactionRequest
        setupMocks  func(*mocks.TransactionRepository, *mocks.BudgetService)
        wantErr     bool
        wantErrType error
        wantResp    CreateTransactionResponse
    }{
        {
            name: "successful_creation",
            req: CreateTransactionRequest{
                FamilyID:    "family_123",
                Amount:      100.50,
                Description: "Test transaction",
                Category:    "groceries",
                Date:        time.Now(),
            },
            setupMocks: func(repo *mocks.TransactionRepository, budget *mocks.BudgetService) {
                budget.EXPECT().GetActiveBudget(mock.Anything, "family_123").Return(&domain.Budget{
                    CategoryLimits: map[string]float64{"groceries": 500.00},
                    Spent:         map[string]float64{"groceries": 200.00},
                }, nil)
                
                repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(t *models.Transaction) bool {
                    return t.Amount == 100.50 && t.Category == "groceries"
                })).Return(&models.Transaction{
                    ID:          "tx_123",
                    Amount:      100.50,
                    Description: "Test transaction",
                }, nil)
            },
            wantErr: false,
            wantResp: CreateTransactionResponse{
                Message: "Transaction created successfully",
            },
        },
        {
            name: "budget_exceeded",
            req: CreateTransactionRequest{
                FamilyID: "family_123",
                Amount:   1000.00,
                Category: "groceries",
            },
            setupMocks: func(repo *mocks.TransactionRepository, budget *mocks.BudgetService) {
                budget.EXPECT().GetActiveBudget(mock.Anything, "family_123").Return(&domain.Budget{
                    CategoryLimits: map[string]float64{"groceries": 500.00},
                    Spent:         map[string]float64{"groceries": 400.00},
                }, nil)
            },
            wantErr:     true,
            wantErrType: &domain.ConflictError{},
        },
        {
            name: "family_not_found",
            req: CreateTransactionRequest{
                FamilyID: "nonexistent",
                Amount:   100.00,
                Category: "groceries",
            },
            setupMocks: func(repo *mocks.TransactionRepository, budget *mocks.BudgetService) {
                budget.EXPECT().GetActiveBudget(mock.Anything, "nonexistent").Return(
                    nil, repository.ErrFamilyNotFound)
            },
            wantErr:     true,
            wantErrType: &domain.NotFoundError{},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := mocks.NewTransactionRepository(t)
            mockBudget := mocks.NewBudgetService(t)
            mockAudit := mocks.NewAuditService(t)
            
            if tt.setupMocks != nil {
                tt.setupMocks(mockRepo, mockBudget)
            }
            
            service := &CreateTransactionService{
                repo:   mockRepo,
                budget: mockBudget,
                audit:  mockAudit,
            }
            
            resp, err := service.Handle(context.Background(), tt.req)
            
            if tt.wantErr {
                require.Error(t, err)
                if tt.wantErrType != nil {
                    assert.ErrorAs(t, err, &tt.wantErrType)
                }
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.wantResp.Message, resp.Message)
                assert.NotNil(t, resp.Transaction)
            }
        })
    }
}
```

#### Integration Testing (HTTP Layer)
```go
func TestCreateTransactionHTTP(t *testing.T) {
    tests := []struct {
        name           string
        method         string
        path           string
        body           interface{}
        headers        map[string]string
        setupMocks     func(*testutils.MockDependencies)
        wantStatusCode int
        wantResponse   func(t *testing.T, body []byte)
        wantErrorCode  string
    }{
        {
            name:   "successful_json_creation",
            method: "POST",
            path:   "/api/v1/transactions",
            body: CreateTransactionRequest{
                FamilyID:    "family_123",
                Amount:      100.50,
                Description: "Test transaction",
                Category:    "groceries",
                Date:        time.Now(),
            },
            headers: map[string]string{
                "Content-Type":  "application/json",
                "Authorization": "Bearer valid_token",
            },
            setupMocks: func(deps *testutils.MockDependencies) {
                deps.BudgetService.EXPECT().GetActiveBudget(mock.Anything, "family_123").Return(validBudget, nil)
                deps.TransactionRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(validTransaction, nil)
            },
            wantStatusCode: http.StatusCreated,
            wantResponse: func(t *testing.T, body []byte) {
                var envelope struct {
                    Data CreateTransactionResponse `json:"data"`
                }
                require.NoError(t, json.Unmarshal(body, &envelope))
                assert.Equal(t, "Transaction created successfully", envelope.Data.Message)
                assert.NotNil(t, envelope.Data.Transaction)
            },
        },
        {
            name:   "xml_request_support",
            method: "POST",
            path:   "/api/v1/transactions",
            body:   `<CreateTransactionRequest><Amount>100.50</Amount><Category>groceries</Category></CreateTransactionRequest>`,
            headers: map[string]string{
                "Content-Type": "application/xml",
                "Accept":       "application/xml",
            },
            setupMocks:     func(deps *testutils.MockDependencies) {},
            wantStatusCode: http.StatusCreated,
        },
        {
            name:   "validation_error",
            method: "POST",
            path:   "/api/v1/transactions",
            body: CreateTransactionRequest{
                Amount:      -100.50, // Invalid: negative amount
                Description: "",       // Invalid: empty description
                Category:    "invalid_category",
            },
            headers: map[string]string{
                "Content-Type": "application/json",
            },
            wantStatusCode: http.StatusBadRequest,
            wantErrorCode:  "VALIDATION_ERROR",
            wantResponse: func(t *testing.T, body []byte) {
                var errorResp ErrorResponse
                require.NoError(t, json.Unmarshal(body, &errorResp))
                assert.Equal(t, "VALIDATION_ERROR", errorResp.Code)
                assert.Contains(t, errorResp.Details, "amount")
                assert.Contains(t, errorResp.Details, "description")
            },
        },
        {
            name:   "unauthorized_request",
            method: "POST",
            path:   "/api/v1/transactions",
            body:   CreateTransactionRequest{},
            headers: map[string]string{
                "Content-Type": "application/json",
                // No Authorization header
            },
            wantStatusCode: http.StatusUnauthorized,
            wantErrorCode:  "UNAUTHORIZED",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup test server with mocked dependencies
            deps := testutils.NewMockDependencies(t)
            if tt.setupMocks != nil {
                tt.setupMocks(deps)
            }
            
            router := setupTestRouterWithDeps(deps)
            server := httptest.NewServer(router)
            defer server.Close()
            
            // Create request
            var bodyReader io.Reader
            if bodyStr, ok := tt.body.(string); ok {
                bodyReader = strings.NewReader(bodyStr)
            } else {
                bodyBytes, _ := json.Marshal(tt.body)
                bodyReader = bytes.NewReader(bodyBytes)
            }
            
            req := httptest.NewRequest(tt.method, tt.path, bodyReader)
            for k, v := range tt.headers {
                req.Header.Set(k, v)
            }
            
            rr := httptest.NewRecorder()
            router.ServeHTTP(rr, req)
            
            // Assertions
            assert.Equal(t, tt.wantStatusCode, rr.Code)
            
            if tt.wantResponse != nil {
                tt.wantResponse(t, rr.Body.Bytes())
            }
            
            if tt.wantErrorCode != "" {
                var errorResp ErrorResponse
                require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &errorResp))
                assert.Equal(t, tt.wantErrorCode, errorResp.Code)
            }
        })
    }
}
```

#### Performance Testing
```go
func BenchmarkCreateTransactionHandler(b *testing.B) {
    service := setupBenchmarkService()
    req := CreateTransactionRequest{
        FamilyID:    "family_123",
        Amount:      100.50,
        Description: "Benchmark transaction",
        Category:    "groceries",
        Date:        time.Now(),
    }
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        ctx := context.Background()
        for pb.Next() {
            _, err := service.Handle(ctx, req)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

## Benefits

### 1. Type Safety
- Compile-time validation of request/response types
- Eliminates runtime JSON marshalling errors
- IDE autocomplete and refactoring support

### 2. Reduced Boilerplate
- Automatic JSON parsing and response encoding
- Standardized error handling
- Built-in observability instrumentation

### 3. Hexagonal Architecture
- Clean separation between HTTP transport and business logic
- Testable business logic without HTTP dependencies
- Support for multiple transport adapters (HTTP, gRPC, CLI)

### 4. Automatic Documentation
- OpenAPI specs generated from handler types
- No documentation drift
- Integrated with existing contract validation

### 5. Enhanced Testing
- Test business logic directly without HTTP mocking
- Separate integration tests for HTTP layer
- Better test coverage and maintainability

### 6. Performance
- Zero runtime overhead from generics
- Compile-time type resolution
- Optional middleware for performance-critical paths

## Developer/Maintainer Perspective Review

### Maintainability Concerns

#### 1. **Cognitive Load**
- **Concern**: Generic syntax `Handler[TRequest, TResponse]` may be intimidating for Go developers unfamiliar with generics
- **Impact**: Slower onboarding, potential resistance to adoption
- **Mitigation**: Provide extensive examples, IDE snippets, and gradual introduction

#### 2. **Debugging Complexity**
- **Concern**: Generic type errors can be cryptic, especially with complex type constraints
- **Impact**: Longer debugging sessions, frustration during development
- **Mitigation**:
    - Custom type aliases for common patterns
    - Better error messages through wrapper types
    - Comprehensive test coverage to catch issues early

#### 3. **Code Generation Dependencies**
- **Concern**: OpenAPI generation through reflection adds build complexity
- **Impact**: Build-time failures, harder to debug schema generation issues
- **Mitigation**:
    - Fallback to manual schema definitions
    - Clear error reporting for schema generation failures
    - Option to disable auto-generation in development

### Development Workflow Impact

#### Positive Impacts
- **Type Safety**: Catch errors at compile time rather than runtime
- **Reduced Boilerplate**: Less repetitive JSON marshalling code
- **Better Testing**: Test business logic without HTTP mocking
- **Consistent Error Handling**: Standardized error mapping across all endpoints

#### Potential Friction Points
- **Learning Curve**: Team needs to understand generics and new patterns
- **IDE Support**: Some IDEs may struggle with complex generic signatures
- **Refactoring**: Changing handler signatures requires updating generic parameters

### Code Review Considerations

#### What Reviewers Should Focus On
```go
// ✅ Good: Clear, simple generic usage
func (s *GetUserService) Handle(ctx context.Context, req GetUserRequest) (GetUserResponse, error) {
    // Business logic here
}

// ❌ Concerning: Complex nested generics
type ComplexHandler[T Validator[K], K Serializable[T], R Response[T, K]] interface {
    Handle(ctx context.Context, req T) (R, error)
}

// ✅ Good: Explicit error handling
if errors.Is(err, repository.ErrNotFound) {
    return GetUserResponse{}, domain.NewNotFoundError("user", req.ID)
}

// ❌ Concerning: Generic error return
if err != nil {
    return GetUserResponse{}, err // Lost error context
}
```

#### Review Checklist
- [ ] Handler has single responsibility
- [ ] Error types are appropriate for HTTP mapping
- [ ] Request/Response types have proper validation tags
- [ ] Business logic is separated from HTTP concerns
- [ ] OpenAPI metadata is meaningful and complete
- [ ] Tests cover both happy path and error cases

### Migration Strategy Refinement

#### Phase 1: Foundation (Weeks 1-2)
```go
// Start with simplest handlers to prove the pattern
r.GET("/health", healthCheckService)  // No complex business logic
r.GET("/metrics", metricsService)     // Read-only operation
```

#### Phase 2: Core Endpoints (Weeks 3-6)
```go
// Migrate CRUD operations one by one
r.GET("/api/v1/transactions/{id}", getTransactionService)
r.POST("/api/v1/transactions", createTransactionService)
// Test extensively before proceeding
```

#### Phase 3: Complex Operations (Weeks 7-12)
```go
// Handle complex business logic and edge cases
r.POST("/api/v1/transactions/bulk", bulkCreateService)
r.GET("/api/v1/reports/budget", budgetReportService)
```

### Operational Concerns

#### Monitoring and Observability
- **Tracing**: Ensure span names are meaningful for generic handlers
- **Metrics**: Handler-specific metrics should include request/response types
- **Logging**: Structured logs should include handler metadata

#### Performance Considerations
- **Memory**: Generic instantiation may increase binary size
- **Compilation**: More complex types may slow build times
- **Runtime**: Zero-cost abstractions should be verified with benchmarks

## Risks and Mitigations

### Risk 1: Learning Curve and Team Adoption
**Impact**: HIGH - Could slow development velocity initially
**Mitigation**:
- Comprehensive documentation with real-world examples
- Pair programming sessions for first implementations
- IDE plugins/snippets for common patterns
- Gradual rollout starting with simple handlers

### Risk 2: Generic Complexity and Debugging
**Impact**: MEDIUM - Could make debugging harder for some developers
**Mitigation**:
- Provide type aliases for common patterns
- Custom error messages for type constraint violations
- Debugging guides specific to generic handlers
- Fallback to non-generic handlers for complex cases

### Risk 3: Build-time Dependencies
**Impact**: MEDIUM - OpenAPI generation could fail builds
**Mitigation**:
- Make OpenAPI generation optional with graceful degradation
- Provide manual schema override capabilities
- Clear error reporting for schema generation issues
- Separate validation for critical build paths

### Risk 4: Performance Regression
**Impact**: LOW-MEDIUM - Generic instantiation could impact performance
**Mitigation**:
- Benchmark existing vs new patterns during migration
- Profile memory usage with complex handlers
- Provide escape hatches for performance-critical paths
- Monitor build times and binary sizes

### Risk 5: Vendor Lock-in to Pattern
**Impact**: LOW - Difficult to migrate away from if issues arise
**Mitigation**:
- Maintain compatibility layer with standard http.HandlerFunc
- Document migration path back to standard handlers
- Keep business logic completely separate from transport layer

## Alternatives Considered

### 1. Code Generation (e.g., oapi-codegen)
**Rejected**: Requires external tools and doesn't integrate with existing TrackedRouter validation.

### 2. Interface{} with Runtime Validation
**Rejected**: Lacks compile-time safety and doesn't reduce boilerplate significantly.

### 3. Decorator Pattern
**Rejected**: More runtime overhead and doesn't provide the same level of type safety.

## Implementation Timeline (Revised)

### Phase 1: Foundation (Weeks 1-3)
- **Week 1**: Core interfaces, basic TypedRouter, simple JSON codec
- **Week 2**: Error mapping system, basic middleware integration
- **Week 3**: Testing framework, documentation, team training

### Phase 2: Validation (Weeks 4-6)
- **Week 4**: Pilot with 2 simple endpoints (health check, get transaction)
- **Week 5**: Team feedback incorporation, debugging improvements
- **Week 6**: Performance benchmarking, production readiness checklist

### Phase 3: Expansion (Weeks 7-10)
- **Week 7-8**: OpenAPI generation, multi-format support
- **Week 9-10**: Migration tooling, automated tests for existing endpoints

### Phase 4: Migration (Weeks 11-16)
- **Week 11-12**: Migrate core CRUD operations
- **Week 13-14**: Complex business logic handlers
- **Week 15-16**: Edge cases, cleanup, final validation

### Success Criteria for Each Phase
- **Phase 1**: Successfully handle basic request/response with type safety
- **Phase 2**: Production deployment of pilot endpoints with monitoring
- **Phase 3**: Complete feature parity with current handler capabilities
- **Phase 4**: All endpoints migrated, performance benchmarks met

## Conclusion

This typed handler abstraction provides a foundation for scalable, maintainable, and type-safe HTTP API development while maintaining backward compatibility and following Go idioms. The design addresses all success criteria while providing a clear migration path and comprehensive testing strategy.

The investment in this abstraction will pay dividends in:
- Reduced development time for new endpoints
- Improved API reliability through type safety
- Automatic documentation generation
- Better testing practices
- Clean architectural separation

## Decision Recommendation

**PROCEED WITH CAUTION** - This ADR presents a well-architected solution that addresses real pain points in our current HTTP handler implementation. However, the complexity and learning curve are non-trivial.

### Recommended Approach
1. **Start Small**: Begin with a pilot implementation on 2-3 simple endpoints
2. **Measure Impact**: Track development velocity, bug rates, and team satisfaction
3. **Iterate Based on Feedback**: Adjust the design based on real-world usage
4. **Provide Escape Hatches**: Always allow fallback to standard handlers for edge cases

### Key Success Factors
- Strong team buy-in and training investment
- Excellent documentation and examples
- Gradual rollout with careful measurement
- Flexibility to adjust course based on experience

## References

- [Go Generics Best Practices](https://go.dev/doc/tutorial/generics)
- [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/)
- [OpenAPI 3.0 Specification](https://swagger.io/specification/)
- [Clean Architecture Principles](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Go Error Handling Best Practices](https://go.dev/blog/go1.13-errors)