# 🏗️ TypedHTTP Fundamentals (15 minutes)

Complete CRUD operations with validation, testing, and production patterns.

## What You'll Learn
- ✅ Full CRUD operations (Create, Read, Update, Delete)
- ✅ Request validation with struct tags
- ✅ Query parameters and pagination
- ✅ Proper error handling with typed errors
- ✅ Comprehensive testing patterns
- ✅ Thread-safe data storage
- ✅ OpenAPI documentation generation

## Quick Start

```bash
# 1. Navigate to fundamentals
cd examples/02-fundamentals

# 2. Run the server
go run main.go

# 3. Try the API endpoints (see below)
```

## API Endpoints

### 📋 List Users
```bash
# Basic listing
curl http://localhost:8080/users

# With pagination
curl "http://localhost:8080/users?limit=1&offset=0"

# With search
curl "http://localhost:8080/users?search=Alice"
```

### 👤 Get User
```bash
curl http://localhost:8080/users/1
```

### ➕ Create User
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Jane Doe","email":"jane@example.com"}'
```

### ✏️ Update User
```bash
# Update name only
curl -X PUT http://localhost:8080/users/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice Updated"}'

# Update email only  
curl -X PUT http://localhost:8080/users/1 \
  -H "Content-Type: application/json" \
  -d '{"email":"alice.new@example.com"}'
```

### 🗑️ Delete User
```bash
curl -X DELETE http://localhost:8080/users/2
```

## Key Features Demonstrated

### 1. Request Validation
```go
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
}
```
- **Automatic validation** before handler execution
- **Custom error responses** for validation failures
- **No manual validation code** needed

### 2. Query Parameters
```go
type ListUsersRequest struct {
    Limit  int    `query:"limit" validate:"omitempty,min=1,max=100" default:"10"`
    Offset int    `query:"offset" validate:"omitempty,min=0" default:"0"`
    Search string `query:"search" validate:"omitempty,min=1,max=50"`
}
```
- **Automatic parsing** from URL query string
- **Default values** when parameters are missing
- **Validation rules** for query parameters

### 3. Proper Error Types
```go
// Not found
return User{}, typedhttp.NewNotFoundError("user", req.ID)

// Conflict (duplicate email)
return User{}, typedhttp.NewConflictError("user with this email already exists")
```
- **Typed errors** instead of generic HTTP errors
- **Consistent error responses** across the API
- **Proper HTTP status codes** automatically

### 4. Thread-Safe Storage
```go
type UserStore struct {
    mu     sync.RWMutex
    users  map[string]User
    nextID int
}
```
- **Concurrent request handling** without data races
- **Read-write locks** for optimal performance
- **Production-ready patterns** for data access

## Testing Patterns

### Direct Handler Testing
```go
func TestGetUserHandler(t *testing.T) {
    store := NewUserStore()
    handler := &GetUserHandler{store: store}
    req := GetUserRequest{ID: "1"}

    resp, err := handler.Handle(context.Background(), req)

    require.NoError(t, err)
    assert.Equal(t, "1", resp.ID)
}
```

### Error Testing
```go
func TestGetUserHandler_NotFound(t *testing.T) {
    // ... setup
    _, err := handler.Handle(context.Background(), req)
    
    require.Error(t, err)
    var notFoundErr *typedhttp.NotFoundError
    assert.ErrorAs(t, err, &notFoundErr)
}
```

### Run the Tests
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...
```

## Code Comparison

### Traditional Go HTTP (50+ lines per endpoint)
```go
func CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", 400)
        return
    }
    
    if req.Name == "" || len(req.Name) < 2 {
        http.Error(w, "Name must be at least 2 characters", 400)
        return
    }
    
    if req.Email == "" || !isValidEmail(req.Email) {
        http.Error(w, "Invalid email", 400)
        return
    }
    
    // Check for duplicates...
    // Create user...
    // Handle errors...
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(201)
    json.NewEncoder(w).Encode(user)
}
```

### TypedHTTP (8 lines per endpoint)
```go
func (h *CreateUserHandler) Handle(ctx context.Context, req CreateUserRequest) (User, error) {
    // Validation automatic via struct tags
    // JSON binding automatic
    // Error responses automatic
    return h.store.Create(req.Name, req.Email)
}
```

**Benefits**: 85% less boilerplate, automatic validation, type safety, easier testing.

## Production Patterns

### Dependency Injection
```go
type CreateUserHandler struct {
    store UserRepository  // Interface for testability
    logger *slog.Logger
    metrics Metrics
}
```

### Error Handling
```go
// Business logic errors become HTTP errors automatically
if duplicateEmail {
    return User{}, typedhttp.NewConflictError("email exists")
}
```

### Testing
```go
// No HTTP server needed for testing
handler := &CreateUserHandler{store: mockStore}
response, err := handler.Handle(ctx, request)
```

## Next Steps

- **[03-intermediate/](../03-intermediate/)** - Middleware, complex validation, and advanced patterns
- **[04-production/](../04-production/)** - Database integration, Docker deployment, monitoring
- **[migration/from-gin/](../migration/from-gin/)** - Migrate existing Gin applications

## Deployment Ready

```bash
# Build for production
go build -o user-api main.go

# Run with Docker
docker build -t user-api .
docker run -p 8080:8080 user-api
```

---

**Ready for more?** → [Next: Production Deployment](../04-production/)