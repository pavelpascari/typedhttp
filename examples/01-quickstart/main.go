package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/pavelpascari/typedhttp/pkg/typedhttp"
)

type GetUserRequest struct {
	ID string `path:"id"`
}

type User struct {
	Name string `json:"name"`
}

type GetUserHandler struct{}

func (h *GetUserHandler) Handle(ctx context.Context, req GetUserRequest) (User, error) {
	return User{Name: "Hello " + req.ID}, nil
}

func main() {
	router := typedhttp.NewRouter()
	handler := &GetUserHandler{}

	typedhttp.GET(router, "/users/{id}", handler)

	fmt.Println("🚀 TypedHTTP Quickstart running on http://localhost:8080")
	fmt.Println("Try: curl http://localhost:8080/users/world")

	log.Fatal(http.ListenAndServe(":8080", router))
}
