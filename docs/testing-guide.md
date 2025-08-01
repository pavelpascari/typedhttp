# TypedHTTP Testing Guide

This guide shows how to test TypedHTTP handlers using the built-in **5/5 Go-idiomatic** test utilities.

## 🎯 **Why Use TypedHTTP Test Utilities?**

- **Eliminates Boilerplate**: No more manual JSON marshaling, header setting, or response parsing
- **Type-Safe**: Leverages Go generics for compile-time type checking
- **Context-Aware**: Full support for timeouts, cancellation, and context propagation
- **Go-Idiomatic**: Follows all Go best practices and conventions
- **Comprehensive**: Covers all HTTP testing scenarios from basic requests to file uploads

## 🚀 **Quick Start Example**

```go
package main_test

import (
    "context"
    "testing"
    "time"

    "github.com/pavelpascari/typedhttp/pkg/testutil"
    "github.com/pavelpascari/typedhttp/pkg/testutil/assert"
    "github.com/pavelpascari/typedhttp/pkg/testutil/client"
    "github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

// Your request/response types
type CreateUserRequest struct {
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=18"`
}

type UserResponse struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

// Your handler
type UserHandler struct{}

func (h *UserHandler) Handle(ctx context.Context, req CreateUserRequest) (UserResponse, error) {
    return UserResponse{
        ID:        "generated-id",
        Name:      req.Name,
        Email:     req.Email,
        CreatedAt: time.Now(),
    }, nil
}

func TestUserHandler(t *testing.T) {
    // Setup TypedHTTP router
    router := typedhttp.NewRouter()
    typedhttp.POST(router, "/users", &UserHandler{})

    // Create test client
    testClient := client.NewClient(router,
        client.WithTimeout(10*time.Second),
    )

    t.Run("create user successfully", func(t *testing.T) {
        // 🎯 Build request with perfect Go idioms
        req := testutil.WithAuth(
            testutil.WithJSON(
                testutil.POST("/users", CreateUserRequest{
                    Name:  "Jane Doe",
                    Email: "jane@example.com",
                    Age:   25,
                }),
            ),
            "auth-token",
        )

        // Execute with context
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        resp, err := testClient.Execute(ctx, req)
        if err != nil {
            t.Fatalf("Request failed: %v", err)
        }

        // Comprehensive assertions
        assert.AssertStatusCreated(t, resp)
        assert.AssertJSONContentType(t, resp)
        assert.AssertJSONField(t, resp, "name", "Jane Doe")
        assert.AssertJSONField(t, resp, "email", "jane@example.com")
        assert.AssertJSONFieldExists(t, resp, "id")
        assert.AssertJSONFieldExists(t, resp, "created_at")
    })
}
```

## 📚 **Testing Patterns**

### 1. **Basic CRUD Operations**

```go
func TestUserCRUD(t *testing.T) {
    router := setupRouter(t)
    client := client.NewClient(router)

    var userID string

    t.Run("create user", func(t *testing.T) {
        req := testutil.POST("/users", CreateUserRequest{
            Name:  "John Doe",
            Email: "john@example.com",
            Age:   30,
        })

        resp, err := client.ExecuteTyped[UserResponse](client, context.Background(), req)
        require.NoError(t, err)

        assert.AssertStatusCreated(t, resp.Response)
        assert.Equal(t, "John Doe", resp.Data.Name)
        userID = resp.Data.ID
    })

    t.Run("get user", func(t *testing.T) {
        req := testutil.WithPathParam(
            testutil.GET("/users/{id}"),
            "id", userID,
        )

        resp, err := client.ExecuteTyped[UserResponse](client, context.Background(), req)
        require.NoError(t, err)

        assert.AssertStatusOK(t, resp.Response)
        assert.Equal(t, userID, resp.Data.ID)
        assert.Equal(t, "John Doe", resp.Data.Name)
    })

    t.Run("update user", func(t *testing.T) {
        req := testutil.WithPathParam(
            testutil.PUT("/users/{id}", UpdateUserRequest{
                Name: "John Smith",
            }),
            "id", userID,
        )

        resp := testutil.MustExecute(t, client, req)
        assert.AssertStatusOK(t, resp)
    })

    t.Run("delete user", func(t *testing.T) {
        req := testutil.WithPathParam(
            testutil.DELETE("/users/{id}"),
            "id", userID,
        )

        resp := testutil.MustExecute(t, client, req)
        assert.AssertStatus(t, resp, 204) // No Content
    })
}
```

### 2. **Authentication Testing**

```go
func TestAuthentication(t *testing.T) {
    client := setupClient(t)

    t.Run("authenticated request succeeds", func(t *testing.T) {
        req := testutil.WithAuth(
            testutil.GET("/protected"),
            "valid-token",
        )

        resp := testutil.MustExecute(t, client, req)
        assert.AssertStatusOK(t, resp)
    })

    t.Run("unauthenticated request fails", func(t *testing.T) {
        req := testutil.GET("/protected")

        resp, err := testutil.ExecuteExpectingError(t, client, req)
        require.NoError(t, err)
        assert.AssertStatusUnauthorized(t, resp)
    })

    t.Run("invalid token fails", func(t *testing.T) {
        req := testutil.WithAuth(
            testutil.GET("/protected"),
            "invalid-token",
        )

        resp, err := testutil.ExecuteExpectingError(t, client, req)
        require.NoError(t, err)
        assert.AssertStatusUnauthorized(t, resp)
    })
}
```

### 3. **Validation Testing**

```go
func TestValidation(t *testing.T) {
    client := setupClient(t)

    tests := []struct {
        name    string
        request CreateUserRequest
        field   string
        error   string
    }{
        {
            name:    "missing name",
            request: CreateUserRequest{Email: "test@example.com", Age: 25},
            field:   "name",
            error:   "required",
        },
        {
            name:    "invalid email",
            request: CreateUserRequest{Name: "John", Email: "invalid", Age: 25},
            field:   "email",
            error:   "email",
        },
        {
            name:    "too young",
            request: CreateUserRequest{Name: "John", Email: "john@example.com", Age: 16},
            field:   "age",
            error:   "min",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := testutil.POST("/users", tt.request)

            resp, err := testutil.ExecuteExpectingError(t, client, req)
            require.NoError(t, err)

            assert.AssertStatusBadRequest(t, resp)
            assert.AssertValidationError(t, resp, tt.field, tt.error)
        })
    }
}
```

### 4. **File Upload Testing**

```go
func TestFileUpload(t *testing.T) {
    client := setupClient(t)

    t.Run("single file upload", func(t *testing.T) {
        fileContent := []byte("test file content")

        req := testutil.WithFile(
            testutil.POST("/upload", map[string]string{
                "name":        "test.txt",
                "description": "Test file upload",
            }),
            "file", fileContent,
        )

        resp, err := client.ExecuteTyped[UploadResponse](client, context.Background(), req)
        require.NoError(t, err)

        assert.AssertStatusCreated(t, resp.Response)
        assert.Equal(t, "test.txt", resp.Data.Filename)
        assert.Equal(t, int64(len(fileContent)), resp.Data.Size)
    })

    t.Run("multiple file upload", func(t *testing.T) {
        req := testutil.WithFiles(
            testutil.POST("/upload-multiple", nil),
            map[string][]byte{
                "file1": []byte("content 1"),
                "file2": []byte("content 2"),
            },
        )

        resp := testutil.MustExecute(t, client, req)
        assert.AssertStatusCreated(t, resp)
        assert.AssertJSONField(t, resp, "count", 2)
    })
}
```

### 5. **Multi-Source Data Testing**

```go
// Test TypedHTTP's multi-source data extraction
func TestMultiSourceData(t *testing.T) {
    client := setupClient(t)

    t.Run("data from multiple sources", func(t *testing.T) {
        req := testutil.WithCookie(
            testutil.WithHeaders(
                testutil.WithQueryParams(
                    testutil.WithPathParams(
                        testutil.GET("/api/{version}/users/{id}"),
                        map[string]string{
                            "version": "v1",
                            "id":      "123",
                        },
                    ),
                    map[string]string{
                        "fields": "id,name,email",
                        "format": "json",
                    },
                ),
                map[string]string{
                    "Authorization": "Bearer token",
                    "X-Request-ID":  "test-123",
                },
            ),
            "session", "session-value",
        )

        resp := testutil.MustExecute(t, client, req)
        assert.AssertStatusOK(t, resp)
        assert.AssertJSONField(t, resp, "id", "123")
    })
}
```

### 6. **Error Handling Testing**

```go
func TestErrorHandling(t *testing.T) {
    client := setupClient(t)

    t.Run("not found error", func(t *testing.T) {
        req := testutil.GET("/users/nonexistent")

        resp, err := testutil.ExecuteExpectingError(t, client, req)
        require.NoError(t, err)

        assert.AssertStatusNotFound(t, resp)
        assert.AssertJSONField(t, resp, "error", "User not found")
    })

    t.Run("internal server error", func(t *testing.T) {
        req := testutil.GET("/users/trigger-error")

        resp, err := testutil.ExecuteExpectingError(t, client, req)
        require.NoError(t, err)

        assert.AssertStatus(t, resp, 500)
        assert.AssertJSONField(t, resp, "error", "Internal server error")
    })
}
```

## 🧪 **Advanced Testing Scenarios**

### 1. **Context and Timeout Testing**

```go
func TestContextHandling(t *testing.T) {
    client := setupClient(t)

    t.Run("request with timeout", func(t *testing.T) {
        ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
        defer cancel()

        req := testutil.GET("/slow-endpoint")

        _, err := client.Execute(ctx, req)
        if err != nil {
            if errors.Is(err, context.DeadlineExceeded) {
                t.Log("Request timed out as expected")
            } else {
                t.Fatalf("Unexpected error: %v", err)
            }
        }
    })

    t.Run("request cancellation", func(t *testing.T) {
        ctx, cancel := context.WithCancel(context.Background())

        go func() {
            time.Sleep(50 * time.Millisecond)
            cancel()
        }()

        req := testutil.GET("/slow-endpoint")
        _, err := client.Execute(ctx, req)

        if errors.Is(err, context.Canceled) {
            t.Log("Request was cancelled as expected")
        }
    })
}
```

### 2. **Concurrent Testing**

```go
func TestConcurrentRequests(t *testing.T) {
    client := setupClient(t)

    t.Run("parallel user creation", func(t *testing.T) {
        const numUsers = 10
        results := make(chan error, numUsers)

        for i := 0; i < numUsers; i++ {
            go func(id int) {
                req := testutil.POST("/users", CreateUserRequest{
                    Name:  fmt.Sprintf("User %d", id),
                    Email: fmt.Sprintf("user%d@example.com", id),
                    Age:   25,
                })

                resp := testutil.MustExecute(t, client, req)
                assert.AssertStatusCreated(t, resp)
                results <- nil
            }(i)
        }

        // Wait for all requests to complete
        for i := 0; i < numUsers; i++ {
            select {
            case err := <-results:
                require.NoError(t, err)
            case <-time.After(5 * time.Second):
                t.Fatal("Request timed out")
            }
        }
    })
}
```

### 3. **Integration with OpenAPI Testing**

```go
func TestOpenAPIGeneration(t *testing.T) {
    router := setupRouter(t)
    client := client.NewClient(router)

    t.Run("openapi spec generation", func(t *testing.T) {
        req := testutil.GET("/openapi.json")

        resp := testutil.MustExecute(t, client, req)
        assert.AssertStatusOK(t, resp)
        assert.AssertJSONContentType(t, resp)
        assert.AssertJSONField(t, resp, "openapi", "3.0.3")
        assert.AssertJSONFieldExists(t, resp, "paths")
        assert.AssertJSONFieldExists(t, resp, "components")
    })
}
```

## 🔧 **Test Setup Helpers**

### Reusable Setup Functions

```go
func setupRouter(t *testing.T) *typedhttp.TypedRouter {
    t.Helper()

    router := typedhttp.NewRouter()
    
    // Register handlers
    typedhttp.GET(router, "/users/{id}", &GetUserHandler{})
    typedhttp.POST(router, "/users", &CreateUserHandler{})
    typedhttp.PUT(router, "/users/{id}", &UpdateUserHandler{})
    typedhttp.DELETE(router, "/users/{id}", &DeleteUserHandler{})
    
    return router
}

func setupClient(t *testing.T) *client.Client {
    t.Helper()

    router := setupRouter(t)
    return client.NewClient(router,
        client.WithTimeout(testutil.DefaultTimeout),
    )
}

func setupAuthenticatedClient(t *testing.T, token string) *client.Client {
    t.Helper()

    baseClient := setupClient(t)
    
    // Return a helper function that adds auth to all requests
    return &authClient{
        client: baseClient,
        token:  token,
    }
}

// Helper wrapper for authenticated requests
type authClient struct {
    client *client.Client
    token  string
}

func (ac *authClient) Execute(ctx context.Context, req testutil.Request) (*testutil.Response, error) {
    authReq := testutil.WithAuth(req, ac.token)
    return ac.client.Execute(ctx, authReq)
}
```

### Database Testing Helpers

```go
func TestWithDatabase(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)

    // Create router with database dependency
    router := setupRouterWithDB(t, db)
    client := client.NewClient(router)

    t.Run("user operations with database", func(t *testing.T) {
        // Create user
        req := testutil.POST("/users", CreateUserRequest{
            Name:  "DB User",
            Email: "db@example.com",
            Age:   25,
        })

        resp, err := client.ExecuteTyped[UserResponse](client, context.Background(), req)
        require.NoError(t, err)

        assert.AssertStatusCreated(t, resp.Response)
        userID := resp.Data.ID

        // Verify user exists in database
        var dbUser User
        err = db.Get(&dbUser, "SELECT * FROM users WHERE id = ?", userID)
        require.NoError(t, err)
        assert.Equal(t, "DB User", dbUser.Name)
    })
}
```

## 📊 **Performance Testing**

```go
func BenchmarkUserCreation(b *testing.B) {
    client := setupClient(&testing.T{}) // Note: This is for example only

    req := testutil.POST("/users", CreateUserRequest{
        Name:  "Benchmark User",
        Email: "bench@example.com",
        Age:   25,
    })

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            resp, err := client.Execute(context.Background(), req)
            if err != nil {
                b.Fatalf("Request failed: %v", err)
            }
            if resp.StatusCode != 201 {
                b.Fatalf("Expected 201, got %d", resp.StatusCode)
            }
        }
    })
}
```

## 🎯 **Best Practices Summary**

1. **Use Context**: Always pass context for timeout and cancellation support
2. **Test Error Cases**: Use `ExecuteExpectingError` for testing error scenarios
3. **Leverage Type Safety**: Use `ExecuteTyped` for type-safe response handling
4. **Build Readable Requests**: Chain request modifiers for clear, readable test setup
5. **Use Specific Assertions**: Prefer specific assertions like `AssertStatusCreated` over generic ones
6. **Setup Helpers**: Create reusable setup functions to reduce test boilerplate
7. **Test Concurrency**: Use Go's testing.T.Parallel() for concurrent test execution
8. **Handle Timeouts**: Use appropriate timeouts for different test scenarios
9. **Validate JSON Structure**: Use dot notation for deep JSON field validation
10. **Follow TDD**: Write tests first, then implement handlers

## 🔗 **Integration with CI/CD**

```yaml
# Example GitHub Actions workflow
name: Test
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    
    - name: Run tests
      run: |
        go test ./... -v -race -coverprofile=coverage.out
        go tool cover -html=coverage.out -o coverage.html
    
    - name: Upload coverage
      uses: actions/upload-artifact@v3
      with:
        name: coverage
        path: coverage.html
```

This testing guide provides everything you need to effectively test TypedHTTP handlers using the **5/5 Go-idiomatic** test utilities. The utilities eliminate boilerplate while maintaining perfect Go conventions and type safety!