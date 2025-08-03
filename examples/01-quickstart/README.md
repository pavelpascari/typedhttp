# 🚀 TypedHTTP Quickstart (5 minutes)

The fastest way to experience TypedHTTP's power.

## What You'll Learn
- ✅ Type-safe HTTP handlers in 18 lines
- ✅ Automatic request/response marshaling
- ✅ Zero configuration needed

## Run It Now

```bash
# 1. Clone and navigate
cd examples/01-quickstart

# 2. Run (requires Go 1.21+)
go run main.go

# 3. Test it out
curl http://localhost:8080/users/world
```

**Expected Response:**
```json
{"name":"Hello world"}
```

## Try These

```bash
# Different users
curl http://localhost:8080/users/alice
curl http://localhost:8080/users/bob

# JSON response every time
curl -v http://localhost:8080/users/developer
```

## What Just Happened?

1. **Type Safety**: The ``req struct{ ID string `path:"id"` }`` automatically extracts `{id}` from the URL
2. **Auto Marshaling**: Your `User` struct becomes JSON automatically  
3. **Zero Boilerplate**: No manual request parsing or response writing

## Compare to Standard Go

**TypedHTTP (25 lines total):**
```go
type GetUserHandler struct{}

func (h *GetUserHandler) Handle(ctx context.Context, req GetUserRequest) (User, error) {
    return User{Name: "Hello " + req.ID}, nil
}
```

**Standard net/http (40+ lines):**
```go
func GetUser(w http.ResponseWriter, r *http.Request) {
    id := mux.Vars(r)["id"]  // Manual extraction
    if id == "" {
        http.Error(w, "Missing ID", 400)
        return
    }
    user := User{Name: "Hello " + id}
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)  // Manual marshaling
    // + error handling, validation, etc.
}
```

## What's Next?

- **[02-fundamentals/](../02-fundamentals/)** - Real CRUD operations with validation and testing
- **[migration/from-gin/](../migration/from-gin/)** - Coming from Gin? See the differences
- **[benchmarks/](../benchmarks/)** - Performance comparison with other frameworks

## Key Benefits Preview

- 🔒 **Type Safety**: Compile-time guarantees for requests/responses
- ⚡ **Performance**: Competitive with Gin/Echo (see benchmarks/)
- 📚 **Auto Docs**: OpenAPI specs generated automatically
- 🧪 **Testable**: Easy to unit test without HTTP mocking
- 🏗️ **Scalable**: Patterns that work for 1 dev or 50+ engineer teams

---

**Ready for more?** → [Next: Real CRUD with Testing](../02-fundamentals/)