# Simple TypedHTTP Example

This example demonstrates the basic usage of the TypedHTTP library for creating type-safe HTTP handlers.

## Running the Example

```bash
cd examples/simple
go run main.go
```

The server will start on port 8080.

## Endpoints

### GET /users/{id}
Retrieves a user by ID.

Example:
```bash
curl http://localhost:8080/users/123
```

Response:
```json
{
  "id": "123",
  "name": "John Doe", 
  "email": "john@example.com",
  "message": "Found user with ID: 123"
}
```

### GET /users/not-found
Demonstrates error handling.

Example:
```bash
curl http://localhost:8080/users/not-found
```

Response (404):
```json
{
  "error": "user with id 'not-found' not found",
  "code": "NOT_FOUND"
}
```

### POST /users
Creates a new user.

Example:
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Jane","email":"jane@example.com"}'
```

Response:
```json
{
  "id": "user_12345",
  "name": "Jane",
  "email": "jane@example.com", 
  "message": "User created successfully"
}
```

### POST /users (conflict)
Demonstrates conflict error handling.

Example:
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Bob","email":"duplicate@example.com"}'
```

Response (409):
```json
{
  "error": "User with this email already exists",
  "code": "CONFLICT"
}
```

## Key Features Demonstrated

1. **Type Safety**: Request and response types are enforced at compile time
2. **Automatic Validation**: Request validation using struct tags
3. **Error Handling**: Structured error responses with appropriate HTTP status codes
4. **OpenAPI Metadata**: Handler documentation through configuration options
5. **Observability**: Built-in support for tracing, metrics, and logging
6. **Clean Architecture**: Business logic separated from HTTP concerns