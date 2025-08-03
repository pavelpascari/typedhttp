# TypedHTTP Recipes

> **Copy-paste solutions** for common API patterns. Each recipe is production-ready with tests.

## 🍳 Quick Recipe Index

| Pattern | Use Case | Files |
|---------|----------|-------|
| **[Authentication](#auth)** | JWT, API keys, session auth | `auth/` |
| **[CRUD Operations](#crud)** | RESTful resource management | `crud/` |
| **[File Uploads](#files)** | Single/multi file handling | `files/` |
| **[Validation](#validation)** | Custom validators, error handling | `validation/` |
| **[Middleware](#middleware)** | Rate limiting, CORS, logging | `middleware/` |
| **[Database Integration](#database)** | SQL, NoSQL, transactions | `database/` |
| **[Error Handling](#errors)** | Custom errors, recovery | `errors/` |
| **[Testing Patterns](#testing)** | Unit, integration, mocking | `testing/` |

---

## 🔐 Authentication Recipes {#auth}

### JWT Authentication
**Use case:** Secure API with JWT tokens

```go
// examples/recipes/auth/jwt.go
type AuthenticatedRequest struct {
    UserID string `header:"Authorization" validate:"required" transform:"jwt_user_id"`
}

type JWTMiddleware struct {
    Secret []byte
}

func (m *JWTMiddleware) Authenticate(token string) (UserID string, error) {
    // JWT validation logic...
}
```

### API Key Authentication  
**Use case:** Simple API key validation

```go
// examples/recipes/auth/apikey.go
type APIKeyRequest struct {
    APIKey string `header:"X-API-Key" validate:"required,len=32"`
}

func ValidateAPIKey(key string) error {
    // API key validation logic...
}
```

### Session-Based Authentication
**Use case:** Traditional session cookies

```go
// examples/recipes/auth/session.go
type SessionRequest struct {
    SessionID string `cookie:"session_id" validate:"required"`
}

func ValidateSession(sessionID string) (*User, error) {
    // Session validation logic...
}
```

[**→ View complete authentication recipes**](auth/)

---

## 📝 CRUD Operations {#crud}

### Basic CRUD Handler
**Use case:** RESTful resource management

```go
// examples/recipes/crud/user_crud.go
type UserResource struct {
    service UserService
}

// GET /users/{id}
func (r *UserResource) Get(ctx context.Context, req GetUserRequest) (User, error) {
    return r.service.GetUser(ctx, req.ID)
}

// POST /users
func (r *UserResource) Create(ctx context.Context, req CreateUserRequest) (User, error) {
    return r.service.CreateUser(ctx, req)
}

// PUT /users/{id}  
func (r *UserResource) Update(ctx context.Context, req UpdateUserRequest) (User, error) {
    return r.service.UpdateUser(ctx, req.ID, req)
}

// DELETE /users/{id}
func (r *UserResource) Delete(ctx context.Context, req DeleteUserRequest) error {
    return r.service.DeleteUser(ctx, req.ID)
}
```

### Pagination & Filtering
**Use case:** List endpoints with pagination

```go
// examples/recipes/crud/pagination.go
type ListUsersRequest struct {
    Page     int      `query:"page" default:"1" validate:"min=1"`
    Limit    int      `query:"limit" default:"20" validate:"min=1,max=100"`
    Sort     string   `query:"sort" default:"created_at" validate:"oneof=name email created_at"`
    Order    string   `query:"order" default:"desc" validate:"oneof=asc desc"`
    Filter   string   `query:"filter"`
    Tags     []string `query:"tags" transform:"comma_split"`
}

type ListUsersResponse struct {
    Users      []User `json:"users"`
    Page       int    `json:"page"`
    Limit      int    `json:"limit"`
    Total      int    `json:"total"`
    TotalPages int    `json:"total_pages"`
}
```

[**→ View complete CRUD recipes**](crud/)

---

## 📁 File Upload Recipes {#files}

### Single File Upload
**Use case:** Profile picture, document upload

```go
// examples/recipes/files/single_upload.go
type UploadFileRequest struct {
    Title       string                `form:"title" validate:"required,min=2,max=100"`
    Description string                `form:"description" validate:"max=500"`
    File        *multipart.FileHeader `form:"file" validate:"required"`
}

func (h *UploadHandler) Handle(ctx context.Context, req UploadFileRequest) (FileResponse, error) {
    // Validate file type and size
    if !isValidFileType(req.File) {
        return FileResponse{}, typedhttp.NewValidationError("Invalid file type")
    }
    
    // Save file logic...
    url, err := h.storage.SaveFile(ctx, req.File)
    return FileResponse{URL: url, Title: req.Title}, err
}
```

### Multiple File Upload
**Use case:** Photo gallery, document batch

```go
// examples/recipes/files/multi_upload.go
type MultiUploadRequest struct {
    Name  string                  `form:"name" validate:"required"`
    Files []*multipart.FileHeader `form:"files" validate:"required,dive,required"`
}

func (h *MultiUploadHandler) Handle(ctx context.Context, req MultiUploadRequest) (MultiFileResponse, error) {
    var savedFiles []FileInfo
    
    for _, file := range req.Files {
        url, err := h.storage.SaveFile(ctx, file)
        if err != nil {
            return MultiFileResponse{}, err
        }
        savedFiles = append(savedFiles, FileInfo{URL: url, Name: file.Filename})
    }
    
    return MultiFileResponse{Files: savedFiles}, nil
}
```

[**→ View complete file upload recipes**](files/)

---

## ✅ Validation Recipes {#validation}

### Custom Validators
**Use case:** Business-specific validation rules

```go
// examples/recipes/validation/custom.go
func init() {
    validator.RegisterValidation("username", validateUsername)
    validator.RegisterValidation("phone", validatePhone)
}

func validateUsername(fl validator.FieldLevel) bool {
    username := fl.Field().String()
    // Username must be 3-20 chars, alphanumeric + underscore
    return regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`).MatchString(username)
}

type CreateAccountRequest struct {
    Username string `json:"username" validate:"required,username"`
    Phone    string `json:"phone" validate:"required,phone"`
}
```

### Conditional Validation
**Use case:** Fields required based on other fields

```go
// examples/recipes/validation/conditional.go
type PaymentRequest struct {
    Method     string `json:"method" validate:"required,oneof=card paypal crypto"`
    CardNumber string `json:"card_number" validate:"required_if=Method card"`
    PayPalID   string `json:"paypal_id" validate:"required_if=Method paypal"`
    CryptoAddr string `json:"crypto_address" validate:"required_if=Method crypto"`
}
```

### Cross-Field Validation
**Use case:** Password confirmation, date ranges

```go
// examples/recipes/validation/cross_field.go
type RegisterRequest struct {
    Password        string `json:"password" validate:"required,min=8"`
    ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
    StartDate       string `json:"start_date" validate:"required,datetime=2006-01-02"`
    EndDate         string `json:"end_date" validate:"required,datetime=2006-01-02,gtfield=StartDate"`
}
```

[**→ View complete validation recipes**](validation/)

---

## 🔧 Middleware Recipes {#middleware}

### Rate Limiting
**Use case:** Protect API from abuse

```go
// examples/recipes/middleware/rate_limit.go
type RateLimitMiddleware struct {
    limiter *rate.Limiter
}

func (m *RateLimitMiddleware) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !m.limiter.Allow() {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### CORS Handler
**Use case:** Cross-origin requests

```go
// examples/recipes/middleware/cors.go
type CORSConfig struct {
    AllowedOrigins []string
    AllowedMethods []string
    AllowedHeaders []string
}

func NewCORSMiddleware(config CORSConfig) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // CORS logic...
            next.ServeHTTP(w, r)
        })
    }
}
```

### Request Logging
**Use case:** Audit trail, debugging

```go
// examples/recipes/middleware/logging.go
type LoggingMiddleware struct {
    logger *slog.Logger
}

func (m *LoggingMiddleware) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        lw := &loggingWriter{ResponseWriter: w}
        next.ServeHTTP(lw, r)
        
        m.logger.Info("request",
            "method", r.Method,
            "path", r.URL.Path,
            "status", lw.status,
            "duration", time.Since(start),
        )
    })
}
```

[**→ View complete middleware recipes**](middleware/)

---

## 🗄️ Database Integration {#database}

### SQL Database (PostgreSQL)
**Use case:** Traditional RDBMS integration

```go
// examples/recipes/database/sql.go
type UserRepository struct {
    db *sql.DB
}

func (r *UserRepository) CreateUser(ctx context.Context, req CreateUserRequest) (User, error) {
    var user User
    err := r.db.QueryRowContext(ctx, `
        INSERT INTO users (name, email, created_at) 
        VALUES ($1, $2, NOW()) 
        RETURNING id, name, email, created_at`,
        req.Name, req.Email,
    ).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
    
    return user, err
}
```

### Transaction Handling
**Use case:** Multi-table operations

```go
// examples/recipes/database/transactions.go
func (s *UserService) CreateUserWithProfile(ctx context.Context, req CreateUserRequest) error {
    return s.db.Transaction(ctx, func(tx *sql.Tx) error {
        // Create user
        userID, err := s.createUser(ctx, tx, req)
        if err != nil {
            return err
        }
        
        // Create profile
        return s.createProfile(ctx, tx, userID, req.Profile)
    })
}
```

### MongoDB Integration
**Use case:** Document database

```go
// examples/recipes/database/mongodb.go
type UserRepository struct {
    collection *mongo.Collection
}

func (r *UserRepository) CreateUser(ctx context.Context, req CreateUserRequest) (User, error) {
    user := User{
        ID:        primitive.NewObjectID(),
        Name:      req.Name,
        Email:     req.Email,
        CreatedAt: time.Now(),
    }
    
    _, err := r.collection.InsertOne(ctx, user)
    return user, err
}
```

[**→ View complete database recipes**](database/)

---

## 🚨 Error Handling {#errors}

### Custom Error Types
**Use case:** Domain-specific errors

```go
// examples/recipes/errors/custom.go
type BusinessError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
}

func (e BusinessError) Error() string {
    return e.Message
}

type ErrorMapper struct{}

func (m *ErrorMapper) MapError(err error) (int, interface{}) {
    switch e := err.(type) {
    case BusinessError:
        return http.StatusBadRequest, e
    case NotFoundError:
        return http.StatusNotFound, map[string]string{"error": e.Error()}
    default:
        return http.StatusInternalServerError, map[string]string{"error": "Internal server error"}
    }
}
```

### Error Recovery
**Use case:** Graceful error handling

```go
// examples/recipes/errors/recovery.go
type RecoveryMiddleware struct{}

func (m *RecoveryMiddleware) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("Panic recovered: %v", err)
                http.Error(w, "Internal server error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

[**→ View complete error handling recipes**](errors/)

---

## 🧪 Testing Patterns {#testing}

### Handler Unit Tests
**Use case:** Test business logic in isolation

```go
// examples/recipes/testing/unit_test.go
func TestUserHandler_CreateUser(t *testing.T) {
    // Setup
    mockService := &MockUserService{}
    handler := &CreateUserHandler{service: mockService}
    
    req := CreateUserRequest{
        Name:  "John Doe",
        Email: "john@example.com",
    }
    
    mockService.On("CreateUser", mock.Anything, req).Return(User{
        ID:    "123",
        Name:  req.Name,
        Email: req.Email,
    }, nil)
    
    // Execute
    user, err := handler.Handle(context.Background(), req)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "John Doe", user.Name)
    mockService.AssertExpectations(t)
}
```

### Integration Tests
**Use case:** Test full HTTP flow

```go
// examples/recipes/testing/integration_test.go
func TestUserAPI_Integration(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    router := setupRouter(db)
    client := testutil.NewClient(router)
    
    t.Run("create and get user", func(t *testing.T) {
        // Create user
        createReq := testutil.WithJSON(
            testutil.POST("/users", CreateUserRequest{
                Name:  "Jane Doe",
                Email: "jane@example.com",
            }),
        )
        
        createResp, err := client.Execute(context.Background(), createReq)
        require.NoError(t, err)
        assert.AssertStatusCreated(t, createResp)
        
        userID := testutil.ExtractJSONField(t, createResp, "id")
        
        // Get user
        getReq := testutil.WithPathParams(
            testutil.GET("/users/{id}"),
            map[string]string{"id": userID},
        )
        
        getResp, err := client.Execute(context.Background(), getReq)
        require.NoError(t, err)
        assert.AssertStatusOK(t, getResp)
        assert.AssertJSONField(t, getResp, "name", "Jane Doe")
    })
}
```

[**→ View complete testing recipes**](testing/)

---

## 🎯 Quick Start with Recipes

1. **Browse the recipe index** above to find your pattern
2. **Copy the recipe files** into your project
3. **Customize** for your specific needs
4. **Run the included tests** to verify functionality

Each recipe includes:
- ✅ **Production-ready code**
- ✅ **Comprehensive tests**
- ✅ **Documentation and examples**
- ✅ **Common variations and extensions**

---

## 📚 Recipe Categories

### **Essential Patterns** (Start here)
- Authentication & Authorization
- CRUD Operations with Pagination
- File Upload Handling
- Error Handling & Recovery

### **Advanced Patterns**
- Database Transactions
- Custom Middleware
- Complex Validation
- Performance Optimization

### **Testing & DevOps**
- Test Utilities & Patterns
- CI/CD Integration
- Monitoring & Observability
- Production Deployment

---

## 🤝 Contributing Recipes

Missing a pattern you need? [**Contribute a recipe!**](../../CONTRIBUTING.md)

Recipe contributions should include:
1. **Working example** with TypedHTTP integration
2. **Comprehensive tests** demonstrating usage
3. **Documentation** explaining the pattern
4. **Common variations** and gotchas

---

**Ready to cook with TypedHTTP?** Pick a recipe and start building! 🍳